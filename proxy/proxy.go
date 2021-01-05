package proxy

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"resizer/manipulator"
	"resizer/media"
	"resizer/registry"
	"resizer/storage"
)

var ErrImageNotFound = errors.New("requested image not found")
var ErrInternalError = errors.New("proxy error")

type ImageProxy interface {
	Proxy(ctx context.Context, dst io.Writer, ID, format, ext string) (*media.Image, *manipulator.Transformation, error)
}

type OnTheFlyPersistingImageProxy struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator manipulator.Manipulator
	parser      *media.Parser
}

// fixme: return transformation
func (p *OnTheFlyPersistingImageProxy) Proxy(
	ctx context.Context,
	dst io.Writer,
	ID, requestedTransformations, ext string,
) (*media.Image, *manipulator.Transformation, error) {
	img, err := p.registry.GetImageByID(ctx, media.ID(ID))
	if err != nil {
		if err == registry.ErrImageNotFound {
			return nil, nil, errors.Wrapf(ErrImageNotFound, "image with ID %v does not exist %v", ID, err)
		}

		return nil, nil, errors.Wrapf(ErrInternalError, "%v", err)
	}

	transformation, err := p.parser.Parse(img, requestedTransformations, ext)
	if err != nil {
		if vErr, ok := err.(*media.ValidationError); ok {
			return nil, nil, &httpError{
				statusCode: 422,
				message: "The given data was invalid",
				details: vErr.Errors(),
			}
		}

		return nil, nil, err
	}

	pr, pw := io.Pipe()
	errCh := make(chan error, 2)
	doneCh := make(chan struct{})

	go func() {
		defer pw.Close()
		if err := p.storage.Download(ctx, pw, img.Bucket, img.OriginalSlice.Filename); err != nil {
			errCh <- errors.Wrapf(
				err,
				"could not download file %s from bucket %s",
				img.OriginalSlice.Filename, img.Bucket)
		}
	}()

	go func() {
		defer func() {
			pr.Close()
			close(doneCh)
		}()

		if transformation.ComputeFilename() == img.OriginalSlice.Filename {
			if _, err := io.Copy(dst, pr); err != nil {
				errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "error copying bytes").Error()}
			}

			return
		}

		if _, err := p.manipulator.Transform(pr, dst, transformation); err != nil {
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "could not transform file").Error()}
		}
	}()

	for {
		select {
			case err := <-errCh:
				if err != nil {
					return nil, nil, err
				}
			case <-doneCh:
				return img, transformation, nil
			case <-ctx.Done():
				return nil, nil, ctx.Err()
		}
	}
}

func NewOnTheFlyPersistingImageProxy(
	r registry.Registry,
	s storage.Storage,
	m manipulator.Manipulator,
	p *media.Parser,
) *OnTheFlyPersistingImageProxy {
	return &OnTheFlyPersistingImageProxy{
		registry:    r,
		storage:     s,
		manipulator: m,
		parser:      p,
	}
}
