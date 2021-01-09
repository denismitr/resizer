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
var ErrSliceNotFound = errors.New("slice not found")
var ErrBadRegistryRequest = errors.New("bad request to registry")
var ErrInternalRegistryError = errors.New("internal registry error")
var ErrEntityNotFound = errors.New("entity not found in registry")
var ErrEntityAlreadyExists = errors.New("entity already exists")

type Registry interface {
	GenerateID() media.ID

	GetImageByID(ctx context.Context, id media.ID) (*media.Image, error)

	// CreateImageWithOriginalSlice - creates a new image
	// along with the first slice, holding storage path for the originally uploaded image
	CreateImageWithOriginalSlice(
		ctx context.Context,
		image *media.Image,
		slice *media.Slice,
	) (media.ID, media.ID, error)

	CreateImage(ctx context.Context, image *media.Image) (media.ID, error)

	CreateSlice(ctx context.Context, slice *media.Slice) (media.ID, error)

	GetImageAndExactMatchSliceIfExists(
		ctx context.Context,
		ID media.ID,
		filename string,
	) (*media.Image, *media.Slice, error)

	GetSliceByImageIDAndFilename(
		ctx context.Context,
		imageID media.ID,
		filename string,
	) (*media.Slice, error)

	Migrate(ctx context.Context) error
}
