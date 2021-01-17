package backoffice

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"resizer/manipulator"
	"resizer/media"
	"resizer/registry"
	"resizer/storage"
	"strings"
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

func (is *ImageService) applyInitialTransformations(useCase *createImageUseCase, errCh chan<- error) <-chan *createImageUseCase {
	resultCh := make(chan *createImageUseCase)

	go func() {
		defer close(resultCh)

		b := &bytes.Buffer{}

		result, err := is.manipulator.Transform(useCase.source, b, nil) // todo: parse and create initial transformation
		if err != nil {
			errCh <- err
			return
		}

		useCase.originalSlice = &createSliceUseCase{
			width:     result.Width,
			height:    result.Height,
			extension: result.Extension,
			filename:  result.OriginalFilename(),
			size:      b.Len(),
		}
		useCase.source = bytes.NewReader(b.Bytes())

		resultCh <- useCase
	}()

	return resultCh
}

func (is *ImageService) saveImageToStorage(
	ctx context.Context,
	useCaseCh <-chan *createImageUseCase,
	errCh chan<- error,
) <-chan *media.Image {
	resultCh := make(chan *media.Image)

	go func() {
		defer close(resultCh)

		useCase := <-useCaseCh
		if useCase == nil {
			return
		}

		img := is.makeNewImage(useCase)

		// fixme: send headers with mime type to the storage !!!
		_, err := is.storage.Put(ctx, img.OriginalSlice.Namespace, img.OriginalSlice.Filename, useCase.source)
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

		_, _, err := is.registry.CreateImageWithOriginalSlice(ctx, img, img.OriginalSlice)
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
		}

		doneCh <- img
	}()

	return doneCh
}

func (is *ImageService) createNewImage(useCase *createImageUseCase) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 2)

	useCaseCh := is.applyInitialTransformations(useCase, errCh)
	imageCh := is.saveImageToStorage(ctx, useCaseCh, errCh)
	doneCh := is.saveNewImageToRegistry(ctx, imageCh, errCh)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
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

func (is *ImageService) makeNewImage(useCase *createImageUseCase) *media.Image {
	var img media.Image
	img.ID = is.registry.GenerateID()
	img.Name = useCase.name
	img.OriginalName = useCase.originalName
	img.OriginalSize = int(useCase.originalSize)
	img.OriginalExt = useCase.originalExt
	img.CreatedAt = time.Now()
	img.UpdatedAt = time.Now()
	img.Namespace = useCase.namespace

	if useCase.publish {
		now := time.Now()
		img.PublishAt = &now
	}

	var slice media.Slice
	slice.ID = is.registry.GenerateID()
	slice.ImageID = img.ID
	slice.Filename = media.ComputeSliceFilename(img.ID, useCase.originalSlice.filename)
	slice.Namespace = img.Namespace
	slice.Path = media.ComputeSlicePath(useCase.namespace, img.ID, useCase.originalSlice.filename)
	slice.Width = useCase.originalSlice.width
	slice.Height = useCase.originalSlice.height
	slice.Extension = useCase.originalSlice.extension
	slice.Size = useCase.originalSlice.size
	slice.IsValid = true
	slice.IsOriginal = true
	slice.Status = media.Active // fixme: processing
	slice.CreatedAt = time.Now()

	img.OriginalSlice = &slice

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

func createURLFriendlyName(useCase *createImageUseCase) string {
	var name string
	if useCase.name != "" {
		name = slug.Make(useCase.name) + "." + useCase.originalExt
	} else {
		segments := strings.Split(useCase.originalName, ".")
		if len(segments) < 2 {
			panic(fmt.Sprintf("how can original name %s not contain extension", useCase.originalName))
		}

		name = slug.Make(strings.Join(segments[:len(segments)-1], ".")) + "." + useCase.originalExt
	}

	return name
}
