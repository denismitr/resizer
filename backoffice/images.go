package backoffice

import (
	"context"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"io"
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
	parser      *media.Parser
}

func NewImages(
	r registry.Registry,
	s storage.Storage,
	m manipulator.Manipulator,
	p *media.Parser,
) *Images {
	return &Images{
		registry:    r,
		storage:     s,
		manipulator: m,
		parser:      p,
	}
}

func (i *Images) createNewImage(useCase *createNewImage) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	errCh := make(chan error, 2)
	transformResultCh := make(chan *manipulator.Result)
	go func() {
		result, err := i.manipulator.Transform(useCase.source, pw, nil)
		if err != nil {
			errCh <- err
			return
		}

		transformResultCh <- result
	}()

	sluggedName := createUrlFriendlyName(useCase)
	useCaseCh := make(chan *createNewImage)
	go func() {
		item, err := i.storage.Put(ctx, useCase.bucket, sluggedName, useCase.source)
		if err != nil {
			errCh <- errors.Wrapf(ErrBackOfficeError, "could not persist image: %v", err)
		}

		transformResult := <-transformResultCh

		useCase.path = item.Path
		useCase.url = item.URL
		useCase.width = transformResult.Width
		useCase.height = transformResult.Height
		useCase.format = transformResult.Format
		useCase.hash = transformResult.Hash

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
	img.Path = useCase.path
	img.Url = useCase.url

	// fixme: do in one transaction
	// fixme: new method in registry CreateImageWithSlice

	if id, err := i.registry.CreateImage(ctx, &img); err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
	} else {
		img.ID = id
	}

	var slice media.Slice
	slice.ImageID = img.ID
	slice.Width = useCase.width
	slice.Height = useCase.height
	slice.Format = useCase.format
	slice.Size = useCase.size
	slice.IsValid = true
	slice.Name = useCase.hash
	slice.CreatedAt = time.Now()

	if id, err := i.registry.CreateSlice(ctx, &slice); err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not create slice in registry: %v", err)
	} else {
		slice.ID = id
		img.Slices = append(img.Slices, slice)
	}

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
