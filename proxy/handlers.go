package proxy

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"resizer/manipulator"
	"time"
)

func makeErrorHandler(err error, lg *logrus.Logger) ErrorHandler {
	return func(rCtx *requestContext) {
		var httpErr *httpError
		if errors.Is(err, ErrResourceNotFound) {
			httpErr = &httpError{statusCode: 404, message: err.Error()}
		} else if errors.Is(err, ErrBadInput) {
			httpErr = &httpError{statusCode: 400, message: err.Error()}
		} else {
			if vErr, ok := err.(*manipulator.ValidationError); ok {
				httpErr = &httpError{
					statusCode: 422,
					message:    "The given data was invalid",
					details:    vErr.Errors(),
				}
			}

			if httpErr, ok := err.(*httpError); ok {
				rCtx.Fail(httpErr)
				return
			}
		}

		if httpErr == nil {
			httpErr = &httpError{statusCode: 500, message: err.Error()}
		}

		if lg != nil {
			lg.Errorln(httpErr.ErrorWithDetails())
		}

		rCtx.Fail(httpErr)
	}
}

func (s *Server) proxyHandler(rCtx *requestContext) error {
	id := rCtx.params[0]
	resizeActions := rCtx.params[1]
	extension := rCtx.params[2]

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	transformation, img, pErr := s.imageProxy.Prepare(ctx, id, resizeActions, extension)
	if pErr != nil {
		return pErr
	}

	// Enable CORS for 3rd party applications
	rCtx.resp.Header().Set("Access-Control-Allow-Origin", "*")

	// Add a Content-Security-Policy to prevent stored-XSS attacks via SVG files
	rCtx.resp.Header().Set("Content-Security-Policy", "script-src 'none'")

	// Disable Content-Type sniffing
	rCtx.resp.Header().Set("X-Content-Type-Options", "nosniff")

	// optimistic headers
	rCtx.resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", resizeActions, extension))
	rCtx.resp.Header().Set("Content-Type", transformation.GetMime())

	return s.imageProxy.Proxy(ctx, rCtx.resp, transformation, img)
}

func createMimeFormExtension(ext string) string {
	if m, ok := mimes[ext]; ok {
		return m
	}

	return "image/jpeg"
}
