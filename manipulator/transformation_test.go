package manipulator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCrop_AllSides(t *testing.T) {
	tr1 := &Transformation{
		Extension: PNG,
		Resize: Resize{
			Crop: Crop{
				Left: 5,
				Right: 5,
				Top: 5,
				Bottom: 5,
			},
		},
	}

	assert.True(t, tr1.RequiresResize())
	assert.True(t, tr1.Resize.Required())
	assert.True(t, tr1.Resize.Crop.Required())
	assert.True(t, tr1.Resize.Crop.AllSides())

	assert.Equal(t, "c5.png", tr1.Filename())

	tr2 := &Transformation{
		Extension: JPEG,
		Resize: Resize{
			Crop: Crop{
				Left: 5,
				Right: 15,
				Top: 5,
				Bottom: 5,
			},
		},
	}

	assert.True(t, tr2.RequiresResize())
	assert.True(t, tr1.Resize.Required())
	assert.True(t, tr2.Resize.Crop.Required())
	assert.False(t, tr2.Resize.Crop.AllSides())

	assert.Equal(t, "cl5_cr15_cr5_cr5.jpg", tr2.Filename())
}
