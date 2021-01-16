package registry

import (
	"context"
	"errors"
	"resizer/media"
)

var ErrCouldNotOpenTx = errors.New("could not open tx")
var ErrRegistryReadFailed = errors.New("registry read error")
var ErrRegistryWriteFailed = errors.New("registry write error")
var ErrBadRegistryRequest = errors.New("bad request to registry")
var ErrInternalRegistryError = errors.New("internal registry error")
var ErrEntityNotFound = errors.New("entity not found in registry")
var ErrEntityAlreadyExists = errors.New("entity already exists")
var ErrInvalidID = errors.New("invalid ID")

type Registry interface {
	GenerateID() media.ID

	GetImageByID(ctx context.Context, id media.ID, onlyPublished bool) (*media.Image, error)
	GetImageWithSlicesByID(ctx context.Context, id media.ID, onlyPublished bool) (*media.Image, error)
	GetImages(ctx context.Context, imageFilter media.ImageFilter) (*media.ImageCollection, error)

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
		onlyPublished bool,
	) (*media.Image, *media.Slice, error)

	GetSliceByImageIDAndFilename(
		ctx context.Context,
		imageID media.ID,
		filename string,
	) (*media.Slice, error)

	Migrate(ctx context.Context) error
	RemoveImageWithAllSlices(ctx context.Context, id media.ID) error
	DepublishImage(ctx context.Context, id media.ID) error
}
