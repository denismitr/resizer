package proxy

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"resizer/manipulator"
	"resizer/media"
	"resizer/registry"
	"resizer/storage"
	"sync"
)

var ErrImageNotFound = errors.New("requested image not found")
var ErrInternalError = errors.New("proxy error")

type ImageProxy interface {
	Proxy(dst io.Writer, ID, format, ext string) (*media.Image, error)
}

type OnTheFlyPersistingImageProxy struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator manipulator.Manipulator
	parser      *media.Parser
}

func (p *OnTheFlyPersistingImageProxy) Proxy(dst io.Writer, ID, requestedTransformations, ext string) (*media.Image, error) {
	ctx := context.Background()
	img, err := p.registry.GetImageByID(ctx, media.ID(ID))
	if err != nil {
		if err == registry.ErrImageNotFound {
			return nil, errors.Wrapf(ErrImageNotFound, "image with ID %v does not exist %v", ID, err)
		}

		return nil, errors.Wrapf(ErrInternalError, "%v", err)
	}

	transformation, err := p.parser.Parse(img, requestedTransformations, ext)
	if err != nil {
		if vErr, ok := err.(*media.ValidationError); ok {
			return nil, &httpError{
				statusCode: 422,
				message: "The given data was invalid",
				details: vErr.Errors(),
			}
		}

		return nil, err
	}

	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := p.storage.Download(ctx, pw, img.Bucket, img.Name); err != nil {
			panic(errors.Wrap(err, "could not download file")) // fixme
		}

		wg.Done()
	}()

	if _, err := p.manipulator.Transform(pr, dst, transformation); err != nil {
		return nil, errors.Wrap(err, "could not transform file")
	}

	wg.Wait()

	return img, nil
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
