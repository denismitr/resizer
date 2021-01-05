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
			Name: "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt: "png",
			OriginalSize: 5678,
			CreatedAt: now,
			UpdatedAt: now,
			PublishAt: nil,
			Bucket: "testbucket",
		}

		id := primitive.NewObjectID()

		record := mapImageToMongoRecord(&img, id)

		assert.Equal(t, imageRecord{
			ID: id,
			Name: "foo_slugged_name",
			OriginalName: "foo name.png",
			OriginalExt: "png",
			OriginalSize: 5678,
			CreatedAt: now,
			UpdatedAt: now,
			PublishAt: nil,
			Bucket: "testbucket",
		}, *record)
	})
}
