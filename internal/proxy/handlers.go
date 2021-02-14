package proxy

import (
	"context"
	"fmt"
	"github.com/denismitr/resizer/internal/media/manipulator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type handler func(*requestContext) error
type errorHandler func(*requestContext)

func makeErrorHandler(err error, lg *logrus.Logger) errorHandler {
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
				rCtx.fail(httpErr)
				return
			}
		}

		if httpErr == nil {
			httpErr = &httpError{statusCode: 500, message: err.Error()}
		}

		if lg != nil {
			lg.Errorln(httpErr.ErrorWithDetails())
		}

		rCtx.fail(httpErr)
	}
}

func makeProxyHandler(imageProxy ImageProxy, lg *logrus.Logger) handler {
	return func(rCtx *requestContext) error {
		id := rCtx.params[0]
		resizeActions := rCtx.params[1]
		extension := rCtx.params[2]

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		transformation, img, pErr := imageProxy.Prepare(ctx, id, resizeActions, extension)
		if pErr != nil {
			return pErr
		}

		rCtx.prepareDownloadHeaders(resizeActions, extension, transformation)

		_, err := imageProxy.Proxy(ctx, rCtx.resp, transformation, img)
		if err != nil {
			return err
		}

		return nil
	}
}

func (c *requestContext) prepareDownloadHeaders(
	resizeActions string,
	extension string,
	transformation *manipulator.Transformation,
) {
	// Enable CORS for 3rd party applications
	c.resp.Header().Set("Access-Control-Allow-Origin", "*")

	// Add a Content-Security-Policy to prevent stored-XSS attacks via SVG files
	c.resp.Header().Set("Content-Security-Policy", "script-src 'none'")

	// Disable Content-Type sniffing
	c.resp.Header().Set("X-Content-Type-Options", "nosniff")

	// optimistic headers
	c.resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", resizeActions, extension))
	c.resp.Header().Set("Content-Type", transformation.GetMime())
}
