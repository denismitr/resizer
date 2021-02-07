package proxy

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

var mimes = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
}

type requestedImage struct {
	imageID string
	file    string
}

type requestContext struct {
	resp   http.ResponseWriter
	req    *http.Request
	params []string
}

func (c *requestContext) Fail(err *httpError) {
	http.Error(c.resp, err.message, err.statusCode)
}

type httpError struct {
	statusCode int
	message    string
	details    map[string]string
}

func (e httpError) Error() string {
	return fmt.Sprintf("[%d] %s", e.statusCode, e.message)
}

func (e httpError) ErrorWithDetails() string {
	return fmt.Sprintf("[%d] %s %v", e.statusCode, e.message, e.details)
}

type Handler func(*requestContext) error
type ErrorHandler func(*requestContext)

type route struct {
	rx      *regexp.Regexp
	handler Handler
}

type Server struct {
	cfg             Config
	logger          *logrus.Logger
	routes          []route
	notFoundHandler ErrorHandler
	imageProxy      ImageProxy

	mu       sync.RWMutex
	mustStop bool
}

func NewServer(cfg Config, logger *logrus.Logger, proxy ImageProxy) *Server {
	server := &Server{
		cfg:             cfg,
		logger:          logger,
		imageProxy:      proxy,
		notFoundHandler: makeErrorHandler(&httpError{statusCode: 404, message: "Route not found"}, logger),
	}

	// todo: allow configuring formats
	server.addRoute(`v1/images/(\w+)/([\w_-]+)\.(png|jpeg|jpg|webp)$`, server.proxyHandler)

	return server
}

func (s *Server) Run(stopCh <-chan os.Signal, shutDownTime time.Duration) error {
	go func() {
		if err := http.ListenAndServe(s.cfg.Port, s); err != nil {
			panic(fmt.Sprintf("shutting down the server %v", err))
		}
	}()

	<-stopCh
	ctx, cancel := context.WithTimeout(context.Background(), shutDownTime)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		s.logger.Errorln(err)
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mustStop = true

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock() // fixme: use atomic
	if s.mustStop {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	defer func() {
		if err := recover(); err != nil {
			s.logger.Errorf("panic recovery: %v", err)
			handler := makeErrorHandler(errors.Errorf("Recovered from panic: %s", err), s.logger)
			handler(&requestContext{resp: w, req: r})
		}
	}()

	rCtx := &requestContext{resp: w, req: r}

	for _, rt := range s.routes {
		matches := rt.rx.FindStringSubmatch(r.URL.Path)
		if len(matches) > 1 {
			rCtx.params = matches[1:]
			if err := rt.handler(rCtx); err != nil {
				makeErrorHandler(err, s.logger)
			}
			return
		}
	}

	s.notFoundHandler(rCtx)
}

func (s *Server) addRoute(pattern string, h Handler) {
	rx := regexp.MustCompile(pattern)
	r := route{rx: rx, handler: h}
	s.routes = append(s.routes, r)
}
