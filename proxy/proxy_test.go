package proxy

import (
	"github.com/stretchr/testify/assert"
	"resizer/manipulator"
	"resizer/media"
	"testing"
)

func Test_createMetadata(t *testing.T) {
	transformation := &manipulator.Transformation{
		Extension: "png",
		Quality: 70,
		Resize: manipulator.Resize{Height: 300, Width: 250},
	}

	transformed := &manipulator.Result{
		Height: 300,
		Width: 250,
		Size: 5005,
	}

	img := &media.Image{
		ID: "fooID",
		OriginalName: "foo_bar_baz.jpg",
		OriginalSlice: &media.Slice{
			Namespace: "my_bucket",
		},
	}

	metadata := createMetadata(img, transformation, transformed)

	assert.Equal(t, "fooID", metadata.imageID)
	assert.Equal(t, "fooID/h300_q70_w250.png", metadata.filename)
	assert.Equal(t, "my_bucket", metadata.namespace)
	assert.Equal(t, 250, metadata.width)
	assert.Equal(t, 300, metadata.height)
	assert.Equal(t, "image/jpeg", metadata.mime)
	assert.Equal(t, "foo_bar_baz.jpg", metadata.originalName)
	assert.Equal(t, 5005, metadata.size)
}
