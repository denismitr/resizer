package proxy

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"sync/atomic"
)

type requestContext struct {
	resp   http.ResponseWriter
	req    *http.Request
	params []string
}

// fail the request context
func (c *requestContext) fail(err *httpError) {
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

// route represents pattern and route handler
type route struct {
	rx      *regexp.Regexp
	handler handler
}

type httpMux struct {
	cfg             Config
	routes          []route
	notFoundHandler errorHandler
	logger          *logrus.Logger
	mustStop        uint32
}

func newMux(cfg Config, imageProxy ImageProxy, lg *logrus.Logger) *httpMux {
	mux := &httpMux{
		cfg:             cfg,
		notFoundHandler: makeErrorHandler(&httpError{statusCode: 404, message: "Route not found"}, lg),
	}

	mux.addRoute(`v1/images/(\w+)/([\w_-]+)\.(png|jpeg|jpg|webp)$`, makeProxyHandler(imageProxy, lg))

	return mux
}

// stop the mux
func (mux *httpMux) stop() {
	atomic.StoreUint32(&mux.mustStop, 1)
}

// addRoute to http mux
func (mux *httpMux) addRoute(pattern string, h handler) {
	rx := regexp.MustCompile(pattern)
	r := route{rx: rx, handler: h}
	mux.routes = append(mux.routes, r)
}

// ServeHTTP implements http.handler
func (mux *httpMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mustStop := atomic.LoadUint32(&mux.mustStop)
	if mustStop == 1 {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			mux.logger.Errorf("panic recovery: %v", err)
			handler := makeErrorHandler(errors.Errorf("Recovered from panic: %s", err), mux.logger)
			handler(&requestContext{resp: w, req: r})
		}
	}()

	rCtx := &requestContext{resp: w, req: r}

	for _, rt := range mux.routes {
		matches := rt.rx.FindStringSubmatch(r.URL.Path)
		if len(matches) > 1 {
			rCtx.params = matches[1:]
			if err := rt.handler(rCtx); err != nil {
				makeErrorHandler(err, mux.logger)
			}
			return
		}
	}

	mux.notFoundHandler(rCtx)
}
