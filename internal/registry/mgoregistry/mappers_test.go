package mgoregistry

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/denismitr/resizer/internal/media"
	"testing"
	"time"
)

func TestMongoRegistry_imageMappers(t *testing.T) {
	t.Run("map image and id to db record", func(t *testing.T) {
		now := time.Now()

		img := media.Image{
			ID:           "5ff9dca506c37f6f5b95cd8a",
			Name:         "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt:  "png",
			OriginalSize: 5678,
			CreatedAt:    now,
			UpdatedAt:    now,
			PublishAt:    nil,
			Namespace:    "testbucket",
		}

		record := mapImageToMongoRecord(&img)
		id, _ := primitive.ObjectIDFromHex("5ff9dca506c37f6f5b95cd8a")

		assert.Equal(t, imageRecord{
			ID:           id,
			Name:         "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt:  "png",
			OriginalSize: 5678,
			CreatedAt:    now,
			UpdatedAt:    now,
			PublishAt:    nil,
			Namespace:    "testbucket",
		}, *record)
	})

	t.Run("map mongo record to image", func(t *testing.T) {
		now := time.Now()
		strID := "5ff9dca506c37f6f5b95cd8a"
		id, _ := primitive.ObjectIDFromHex(strID)

		record := imageRecord{
			ID:           id,
			Name:         "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt:  "png",
			OriginalSize: 5678,
			CreatedAt:    now,
			UpdatedAt:    now,
			PublishAt:    nil,
			Namespace:    "testbucket",
		}

		img := mapMongoRecordToImage(&record)

		assert.Equal(t, &media.Image{
			ID:           media.ID(strID),
			Name:         "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt:  "png",
			OriginalSize: 5678,
			CreatedAt:    now,
			UpdatedAt:    now,
			PublishAt:    nil,
			Namespace:    "testbucket",
		}, img)
	})
}

func TestMongoRegistry_sliceMappers(t *testing.T) {
	t.Run("map image and id to db record", func(t *testing.T) {
		now := time.Now()

		sl := media.Slice{
			ID:         "5ff9dca506c37f6f5b95cd8a",
			ImageID:    "3aa6dea106c37a4f6b01cd5b",
			Height:     100,
			Width:      200,
			Status:     media.Pending,
			Namespace:  "mybucket",
			Path:       "mycollection/601ee07d9c573e3bfb65cb4b/h413_w335.png",
			Extension:  "png",
			IsOriginal: true,
			IsValid:    true,
			Size:       5500,
			CreatedAt:  now,
		}

		record := mapSliceToMongoRecord(&sl)
		id, _ := primitive.ObjectIDFromHex("5ff9dca506c37f6f5b95cd8a")
		imageID, _ := primitive.ObjectIDFromHex("3aa6dea106c37a4f6b01cd5b")

		assert.Equal(t, sliceRecord{
			ID:         id,
			ImageID:    imageID,
			Height:     100,
			Width:      200,
			Size:       5500,
			Status:     string(media.Pending),
			Namespace:  "mybucket",
			Path:       "mycollection/601ee07d9c573e3bfb65cb4b/h413_w335.png",
			Extension:  "png",
			IsOriginal: true,
			IsValid:    true,
			CreatedAt:  now,
		}, *record)
	})
}
