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

	if err := p.s.Download(ctx, dst, img.Bucket, img.Name); err != nil {
		return nil, errors.Wrap(err, "could not download file")
	}

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
