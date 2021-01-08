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
	manipulator manipulator.Manipulator
	parser      *manipulator.Parser
}

func NewImages(
	r registry.Registry,
	s storage.Storage,
	m manipulator.Manipulator,
	p *manipulator.Parser,
) *Images {
	return &Images{
		registry:    r,
		storage:     s,
		manipulator: m,
		parser:      p,
	}
}

type transformedImage struct {
	img   *manipulator.TransformationResult
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

	useCaseCh := make(chan *createNewImage)
	go func() {
		transformed := <-transformedCh
		// fixme: send headers with mime type to the storage
		item, err := i.storage.Put(ctx, useCase.bucket, transformed.img.Filename, bytes.NewReader(transformed.bytes))
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not persist image: %v", err)
			return
		}

		useCase.originalSlice = &createNewSlice{
			path:      item.Path,
			width:     transformed.img.Width,
			height:    transformed.img.Height,
			extension: transformed.img.Extension,
			filename:  transformed.img.Filename,
			size:      len(transformed.bytes),
		}

		useCaseCh <- useCase
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errCh:
			return nil, err
		case uc := <-useCaseCh:
			return i.saveNewImageToRegistry(ctx, uc)
		}
	}
}

func (i *Images) saveNewImageToRegistry(
	ctx context.Context,
	useCase *createNewImage,
) (*media.Image, error) {
	sluggedName := createUrlFriendlyName(useCase)

	var img media.Image
	img.Name = sluggedName
	img.OriginalName = useCase.originalName
	img.OriginalSize = int(useCase.originalSize)
	img.OriginalExt = useCase.originalExt
	img.PublishAt = nil
	img.CreatedAt = time.Now()
	img.UpdatedAt = time.Now()
	img.Bucket = useCase.bucket

	var slice media.Slice
	slice.Path = useCase.bucket + "/" + useCase.originalSlice.filename
	slice.Filename = useCase.originalSlice.filename
	slice.Width = useCase.originalSlice.width
	slice.Height = useCase.originalSlice.height
	slice.Extension = useCase.originalSlice.extension
	slice.Size = useCase.originalSlice.size
	slice.Bucket = useCase.bucket
	slice.IsValid = true
	slice.IsOriginal = true
	slice.CreatedAt = time.Now()

	imageID, sliceID, err := i.registry.CreateImageWithOriginalSlice(ctx, &img, &slice)
	if err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
	}

	img.ID = imageID
	slice.ID = sliceID
	slice.ImageID = imageID
	img.OriginalSlice = &slice

	return &img, nil
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
