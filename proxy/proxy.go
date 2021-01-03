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
	r registry.Registry
	s storage.Storage
	m manipulator.Manipulator
}

func (p *OnTheFlyPersistingImageProxy) Proxy(dst io.Writer, ID, format, ext string) (*media.Image, error) {
	ctx := context.Background()
	img, err := p.r.GetImageByID(ctx, media.ID(ID))
	if err != nil {
		if err == registry.ErrImageNotFound {
			return nil, errors.Wrapf(ErrImageNotFound, "image with ID %v does not exist %v", ID, err)
		}

		return nil, errors.Wrapf(ErrInternalError, "%v", err)
	}

	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if err := p.s.Download(ctx, pw, img.Bucket, img.Name); err != nil {
			panic(errors.Wrap(err, "could not download file")) // fixme
		}

		wg.Done()
	}()

	if err := p.m.Transform(pr, dst, &manipulator.Transformation{
		Resize: manipulator.Resize{
			Height: 100,
		},
		Flip: manipulator.Flip{Vertical: true},
		Quality: 90,
		Format: manipulator.PNG,
	}); err != nil {
		return nil, errors.Wrap(err, "could not transform file")
	}

	wg.Wait()

	return img, nil
}

func NewOnTheFlyPersistingImageProxy(
	r registry.Registry,
	s storage.Storage,
	m manipulator.Manipulator,
) *OnTheFlyPersistingImageProxy {
	return &OnTheFlyPersistingImageProxy{
		r: r,
		s: s,
		m: m,
	}
}
