package registry

import (
	"context"
	"errors"
	"resizer/media"
)

var ErrCouldNotOpenTx = errors.New("could not open tx")
var ErrTxFailed = errors.New("tx failed")
var ErrRegistryReadFailed = errors.New("registry read error")
var ErrRegistryWriteFailed = errors.New("registry write error")
var ErrImageNotFound = errors.New("image not found")

type Registry interface {
	GetImageByID(ctx context.Context, id media.ID) (*media.Image, error)
	CreateImage(ctx context.Context, image *media.Image) (media.ID, error)
}
