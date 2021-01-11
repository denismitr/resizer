package mgoregistry

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"resizer/media"
	"testing"
	"time"
)

func TestMongoRegistry_mapImageToMongoRecord(t *testing.T) {
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
}
