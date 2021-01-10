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
	"time"
)

var ErrBackOfficeError = errors.New("back office error")

// Images is a collection of use cases specific to the back office
// handling business logic for processing images
type Images struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator *manipulator.Manipulator
}

func NewImages(
	r registry.Registry,
	s storage.Storage,
	m *manipulator.Manipulator,
) *Images {
	return &Images{
		registry:    r,
		storage:     s,
		manipulator: m,
	}
}

type transformedImage struct {
	img   *manipulator.Result
	bytes []byte
}

func (i *Images) createNewImage(useCase *createNewImage) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 2)
	transformedCh := make(chan *transformedImage)
	go func() {
		b := &bytes.Buffer{}
		result, err := i.manipulator.Transform(useCase.source, b, nil)
		if err != nil {
			errCh <- err
			return
		}

		transformedCh <- &transformedImage{img: result, bytes: b.Bytes()}
	}()

	imageCh := make(chan *media.Image)
	go func() {
		transformed := <-transformedCh

		useCase.originalSlice = &createNewSlice{
			width:     transformed.img.Width,
			height:    transformed.img.Height,
			extension: transformed.img.Extension,
			filename:  transformed.img.OriginalFilename(), // fixme
			size:      len(transformed.bytes),
		}

		img := i.createImage(useCase)

		// fixme: send headers with mime type to the storage !!!
		_, err := i.storage.Put(ctx, img.OriginalSlice.Bucket, img.OriginalSlice.Filename, bytes.NewReader(transformed.bytes))
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not persist image: %v", err)
			return
		}

		imageCh <- img
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			return nil, err
		case img := <-imageCh:
			return i.saveNewImageToRegistry(ctx, img)
		}
	}
}

func (i *Images) createImage(useCase *createNewImage) *media.Image {
	var img media.Image
	img.ID = i.registry.GenerateID()
	img.Name = useCase.name
	img.OriginalName = useCase.originalName
	img.OriginalSize = int(useCase.originalSize)
	img.OriginalExt = useCase.originalExt
	img.CreatedAt = time.Now()
	img.UpdatedAt = time.Now()
	img.Bucket = useCase.bucket

	if useCase.publish {
		now := time.Now()
		img.PublishAt = &now
	}

	var slice media.Slice
	slice.ID = i.registry.GenerateID()
	slice.ImageID = img.ID
	slice.Filename = media.ComputeSliceFilename(img.ID, useCase.originalSlice.filename)
	slice.Bucket = img.Bucket
	slice.Path = media.ComputeSlicePath(useCase.bucket, img.ID, useCase.originalSlice.filename)
	slice.Width = useCase.originalSlice.width
	slice.Height = useCase.originalSlice.height
	slice.Extension = useCase.originalSlice.extension
	slice.Size = useCase.originalSlice.size
	slice.IsValid = true
	slice.IsOriginal = true
	slice.Status = media.Ready // fixme: processing
	slice.CreatedAt = time.Now()

	img.OriginalSlice = &slice

	return &img
}

func (i *Images) saveNewImageToRegistry(
	ctx context.Context,
	img *media.Image,
) (*media.Image, error) {
	_, _, err := i.registry.CreateImageWithOriginalSlice(ctx, img, img.OriginalSlice)
	if err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
	}

	return img, nil
}

func createUrlFriendlyName(useCase *createNewImage) string {
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
