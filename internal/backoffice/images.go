package backoffice

import (
	"bytes"
	"context"
	"github.com/denismitr/resizer/internal/media"
	"github.com/denismitr/resizer/internal/media/manipulator"
	"github.com/denismitr/resizer/internal/registry"
	"github.com/denismitr/resizer/internal/storage"
	"github.com/pkg/errors"
	"io"
	"sync"
	"time"
)

var ErrBackOfficeError = errors.New("back office error")
var ErrResourceNotFound = errors.New("resource not found")

// ImageService is a collection of use cases specific to the back office
// handling business logic for processing images
type ImageService struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator *manipulator.Manipulator
}

func NewImageService(
	r registry.Registry,
	s storage.Storage,
	m *manipulator.Manipulator,
) *ImageService {
	return &ImageService{
		registry:    r,
		storage:     s,
		manipulator: m,
	}
}

func (is *ImageService) getImages(filter media.ImageFilter) (*media.ImageCollection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection, err := is.registry.GetImages(ctx, filter)
	if err != nil {
		return nil, err
	}

	return collection, nil
}

func (is *ImageService) createOriginalSlice(source io.Reader, newImage *media.Image, errCh chan<- error) <-chan *originalSlice {
	resultCh := make(chan *originalSlice)

	go func() {
		defer close(resultCh)

		b := &bytes.Buffer{}
		slice, err := is.manipulator.CreateOriginalSlice(source, b, newImage) // todo: parse and create initial transformation
		if err != nil {
			errCh <- err
			return
		}

		resultCh <- &originalSlice{
			slice: slice,
			content: bytes.NewReader(b.Bytes()),
		}
	}()

	return resultCh
}

func (is *ImageService) saveOriginalSliceToStorage(
	ctx context.Context,
	img *media.Image,
	originalSliceCh <-chan *originalSlice,
	errCh chan<- error,
) <-chan *media.Image {
	resultCh := make(chan *media.Image)

	go func() {
		defer close(resultCh)

		os := <-originalSliceCh
		if os == nil {
			return
		}

		img.OriginalSlice = os.slice

		// fixme: send headers with mime type to the storage !!!
		_, err := is.storage.Put(ctx, img.OriginalSlice.Namespace, img.OriginalSlice.Filename, os.content)
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not persist image: %v", err)
			return
		}

		resultCh <- img
	}()

	return resultCh
}

func (is *ImageService) saveNewImageToRegistry(
	ctx context.Context,
	imageCh <-chan *media.Image,
	errCh chan<- error,
) <-chan *media.Image {
	doneCh := make(chan *media.Image)

	go func() {
		defer close(doneCh)

		img, ok := <-imageCh
		if img == nil || !ok {
			return
		}

		img.OriginalSlice.ID = is.registry.GenerateID()
		img.OriginalSlice.Status = media.Active

		_, _, err := is.registry.CreateImageWithOriginalSlice(ctx, img, img.OriginalSlice)
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
		}

		doneCh <- img
	}()

	return doneCh
}

func (is *ImageService) createNewImage(dto *createImageDTO) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	img := is.makeNewImage(dto)

	errCh := make(chan error, 2)

	originalSliceCh := is.createOriginalSlice(dto.source, img, errCh)
	imageCh := is.saveOriginalSliceToStorage(ctx, img, originalSliceCh, errCh)
	doneCh := is.saveNewImageToRegistry(ctx, imageCh, errCh)

	for {
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "could not create new image")
		case err := <-errCh:
			return nil, err
		case img, ok := <-doneCh:
			if img == nil || !ok {
				return nil, errors.New("something went wrong")
			}

			return img, nil
		}
	}
}

func (is *ImageService) makeNewImage(dto *createImageDTO) *media.Image {
	var img media.Image
	img.ID = is.registry.GenerateID()
	img.Name = dto.name
	img.OriginalName = dto.originalName
	img.OriginalSize = int(dto.originalSize)
	img.OriginalExt = dto.originalExt
	img.CreatedAt = time.Now()
	img.UpdatedAt = time.Now()
	img.Namespace = dto.namespace

	if dto.publish {
		now := time.Now()
		img.PublishAt = &now
	}

	return &img
}

func (is *ImageService) getImage(id string) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	img, err := is.registry.GetImageWithSlicesByID(ctx, media.ID(id), false)
	if err != nil {
		if errors.Is(err, registry.ErrEntityNotFound) {
			return nil, errors.Wrapf(ErrResourceNotFound, "%s", err.Error())
		}

		return nil, err
	}

	return img, nil
}

func (is *ImageService) removeImage(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	img, err := is.registry.GetImageWithSlicesByID(ctx, media.ID(id), false)
	if err != nil {
		if errors.Is(err, registry.ErrEntityNotFound) {
			return ErrResourceNotFound
		}

		return err
	}

	if err := is.registry.DepublishImage(ctx, media.ID(id)); err != nil {
		if errors.Is(err, registry.ErrEntityNotFound) {
			return ErrResourceNotFound
		}

		return err
	}

	errCh := make(chan error, len(img.Slices))
	doneRemoveFromStorage := is.removeFromStorage(ctx, img.Slices, errCh)

	for {
		select {
			case err := <-errCh:
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-doneRemoveFromStorage:
				return is.registry.RemoveImageWithAllSlices(ctx, media.ID(id))
		}
	}
}

func (is *ImageService) removeFromStorage(
	ctx context.Context,
	slices media.Slices,
	errCh chan<- error,
) <-chan struct{} {
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		var wg sync.WaitGroup
		for _, slice := range slices {
			wg.Add(1)
			go func(namespace, filename string) {
				defer wg.Done()

				if err := is.storage.Remove(ctx, namespace, filename); err != nil {
					errCh <- err
				}
			}(slice.Namespace, slice.Filename)
		}

		wg.Wait()
	}()

	return doneCh
}
