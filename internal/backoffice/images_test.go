package backoffice

import (
	"github.com/denismitr/resizer/internal/media"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_makeNewImage(t *testing.T) {
	id := media.ID("myid")
	now := time.Now()

	t.Run("it can create image without immediate publication", func(t *testing.T) {
		createImage := createImageDTO{
			name: "foo",
			originalName: "foo_original",
			originalExt: "png",
			originalSize: 5600,
			namespace: "bucketFoo",
		}

		img := makeNewImage(id, &createImage, now)

		assert.Equal(t, img.Namespace, createImage.namespace)
		assert.Equal(t, img.Name, createImage.name)
		assert.Equal(t, img.OriginalName, createImage.originalName)
		assert.Equal(t, img.OriginalExt, createImage.originalExt)
		assert.Equal(t, img.OriginalSize, int(createImage.originalSize)) // fixme
		assert.Equal(t, img.CreatedAt, now)
		assert.Equal(t, img.UpdatedAt, now)

		assert.Nil(t, img.PublishAt)
	})

	t.Run("it can create image with immediate publication", func(t *testing.T) {
		createImage := createImageDTO{
			name: "foo",
			originalName: "foo_original",
			originalExt: "png",
			originalSize: 5600,
			publish: true,
			namespace: "bucketFoo",
		}

		img := makeNewImage(id, &createImage, now)

		assert.Equal(t, img.Namespace, createImage.namespace)
		assert.Equal(t, img.Name, createImage.name)
		assert.Equal(t, img.OriginalName, createImage.originalName)
		assert.Equal(t, img.OriginalExt, createImage.originalExt)
		assert.Equal(t, img.OriginalSize, int(createImage.originalSize)) // fixme
		assert.Equal(t, img.CreatedAt, now)
		assert.Equal(t, img.UpdatedAt, now)

		assert.Equal(t, img.PublishAt, &now)
	})
}
