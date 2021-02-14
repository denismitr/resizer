package manipulator

import (
	"github.com/denismitr/resizer/internal/media"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCrop_AllSides(t *testing.T) {
	t.Run("crop all sides", func(t *testing.T) {
		tr := &Transformation{
			Extension: media.PNG,
			Resize: Resize{
				Crop: Crop{
					Left: 5,
					Right: 5,
					Top: 5,
					Bottom: 5,
				},
			},
		}

		assert.True(t, tr.RequiresResize())
		assert.True(t, tr.Resize.Required())
		assert.True(t, tr.Resize.Crop.Required())
		assert.True(t, tr.Resize.Crop.AllSides())

		assert.Equal(t, "c5.png", tr.Filename())
	})

	t.Run("crop differently by side", func(t *testing.T) {
		tr := &Transformation{
			Extension: media.JPEG,
			Resize: Resize{
				Crop: Crop{
					Left: 5,
					Right: 15,
					Top: 5,
					Bottom: 5,
				},
			},
		}

		assert.True(t, tr.RequiresResize())
		assert.True(t, tr.Resize.Required())
		assert.True(t, tr.Resize.Crop.Required())
		assert.False(t, tr.Resize.Crop.AllSides())

		assert.Equal(t, "cl5_cr15_cr5_cr5.jpg", tr.Filename())
	})
}

func TestCrop_Resize(t *testing.T) {
	t.Run("only height", func(t *testing.T) {
		tr := &Transformation{
			Extension: media.PNG,
			Resize: Resize{
				Height: 300,
			},
		}

		assert.True(t, tr.RequiresResize())
		assert.True(t, tr.Resize.Required())
		assert.False(t, tr.Resize.Crop.Required())
		assert.True(t, tr.Resize.Crop.None())
		assert.False(t, tr.Resize.Crop.AllSides())

		assert.Equal(t, "h300.png", tr.Filename())
	})

	t.Run("only width", func(t *testing.T) {
		tr := &Transformation{
			Extension: media.JPEG,
			Resize: Resize{
				Width: 450,
			},
		}

		assert.True(t, tr.RequiresResize())
		assert.True(t, tr.Resize.Required())
		assert.False(t, tr.Resize.Crop.Required())
		assert.False(t, tr.Resize.Crop.AllSides())

		assert.Equal(t, "w450.jpg", tr.Filename())
	})
}