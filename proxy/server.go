package proxy

import (
	"context"
	"encoding/json"
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

type Handler func(*requestContext) *httpError
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
	proxy           ImageProxy

	mu       sync.RWMutex
	mustStop bool
}

func NewServer(cfg Config, logger *logrus.Logger, proxy ImageProxy) *Server {
	server := &Server{
		cfg:             cfg,
		logger:          logger,
		proxy:           proxy,
		notFoundHandler: errorHandler(404, "Route not found", nil),
	}

	// todo: allow configuring formats
	server.addRoute(`v1/images/(\w+)/([\w_-]+)\.(png|jpeg|jpg|webp)$`, server.fetchImage)

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
	s.mu.RLock()
	if s.mustStop {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	// todo: panic recovery

	rCtx := &requestContext{resp: w, req: r}

	for _, rt := range s.routes {
		matches := rt.rx.FindStringSubmatch(r.URL.Path)
		if len(matches) > 1 {
			rCtx.params = matches[1:]
			if err := rt.handler(rCtx); err != nil {
				s.logger.Errorln(err.ErrorWithDetails())
				errorHandler(err.statusCode, err.message, err.details)(rCtx)
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

func errorHandler(status int, message string, details map[string]string) ErrorHandler {
	return func(rCtx *requestContext) { // fixme
		rCtx.resp.WriteHeader(status)

		accept := rCtx.req.Header.Get("Accept")
		if accept == "application/json" {
			rCtx.resp.Header().Set("Content-Type", "application/json")
			b, err := json.Marshal(map[string]string{"message": message})
			if err != nil {
				panic("How? " + err.Error())
			}

			if _, err := rCtx.resp.Write(b); err != nil {
				panic("How? " + err.Error()) // fixme: log and leave
			}

			return
		}

		rCtx.resp.Header().Set("Content-Type", "text/plain")
		if _, err := rCtx.resp.Write([]byte(message)); err != nil {
			panic("How? " + err.Error()) // fixme: log and leave
		}

		return
	}
}

func (s *Server) fetchImage(rCtx *requestContext) *httpError {
	id := rCtx.params[0]
	resizeActions := rCtx.params[1]
	extension := rCtx.params[2]

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	metadata, err := s.proxy.Proxy(ctx, rCtx.resp, id, resizeActions, extension)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			return &httpError{statusCode: 404, message: err.Error()} // fixme
		}

		if httpErr, ok := err.(*httpError); ok {
			return httpErr
		}

		return &httpError{statusCode: 500, message: err.Error()} // fixme
	}

	//rCtx.resp.WriteHeader(200)
	rCtx.resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", metadata.filename))
	rCtx.resp.Header().Set("Content-Type", fmt.Sprintf("image/%s", metadata.mime))
	rCtx.resp.Header().Set("Original-Name", metadata.originalName)

	return nil
}

func createMimeFormExtension(ext string) string {
	if m, ok := mimes[ext]; ok {
		return m
	}

	return "image/jpeg"
}
