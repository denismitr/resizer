package backoffice

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/gosimple/slug"
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
	R registry.Registry
	S storage.Storage
}

func (i *Images) createNewImage(useCase createNewImage) (*media.Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sluggedName := createUrlFriendlyName(useCase)

	item, err := i.S.Put(ctx, useCase.bucket, sluggedName, useCase.source)
	if err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not persist image: %v", err)
	}

	var img media.Image
	img.OriginalName = useCase.originalName
	img.OriginalSize = int(useCase.originalSize)
	img.OriginalExt = useCase.originalExt
	img.PublishAt = time.Time{} // fixme
	img.CreatedAt = time.Now()
	img.UpdatedAt = time.Now()
	img.Bucket = useCase.bucket
	img.Path = item.Path
	img.Url = item.URL

	if id, err := i.R.CreateImage(ctx, &img); err != nil {
		return nil, errors.Wrapf(ErrBackOfficeError, "could not create image in registry: %v", err)
	} else {
		img.ID = id
	}

	return &img, nil
}

func createUrlFriendlyName(useCase createNewImage) string {
	var name string
	if useCase.name != "" {
		name = slug.Make(useCase.name) + "." + useCase.originalExt
	} else {
		segments := strings.Split(useCase.originalName, ".")
		if len(segments) < 2 {
			panic(fmt.Sprintf("how can original name %s not contain extension", useCase.originalName))
		}

		name = slug.Make(strings.Join(segments[:len(segments)-2], ".")) + "." + useCase.originalExt
	}

	return name
}
