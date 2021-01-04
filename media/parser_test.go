package media

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"resizer/manipulator"
	"testing"
)

func Test_createTransformation(t *testing.T) {
	tt := []struct{
		requestedTransformations string
		extension string
		img *Image
		err error
		expected *manipulator.Transformation
		filename string
	}{
		{
			requestedTransformations: "h200",
			extension: "jpg",
			img: &Image{},
			expected: &manipulator.Transformation{
				Resize: manipulator.Resize{Height: 200},
				Format: manipulator.JPEG,
			},
			filename: "h200.jpg",
		},
		{
			requestedTransformations: "h200_w400",
			extension: "png",
			img: &Image{},
			expected: &manipulator.Transformation{
				Resize: manipulator.Resize{Height: 200, Width: 400},
				Format: manipulator.PNG,
			},
			filename: "h200_w400.png",
		},
		{
			requestedTransformations: "h200_w400_q80",
			extension: "png",
			img: &Image{},
			expected: &manipulator.Transformation{
				Resize: manipulator.Resize{Height: 200, Width: 400},
				Quality: 80,
				Format: manipulator.PNG,
			},
			filename: "h200_q80_w400.png",
		},
		{
			requestedTransformations: "h200_w400_q80_p50",
			extension: "png",
			img: &Image{},
			expected: &manipulator.Transformation{
				Resize: manipulator.Resize{Height: 200, Width: 400, Proportion: 50},
				Quality: 80,
				Format: manipulator.PNG,
			},
			filename: "h200_p50_q80_w400.png",
		},
	}

	p := NewParser()

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%storage-%storage", tc.requestedTransformations, tc.extension), func(t *testing.T) {
			transformation, err := p.Parse(tc.img, tc.requestedTransformations, tc.extension)
			if tc.err != nil {
				assert.Error(t, err)
				return
			}

			if ! assert.NoError(t, err) {
				t.Fatal(err)
			}

			assert.NotNil(t, transformation)
			assert.Equal(t, tc.expected, transformation)
			assert.Equal(t, tc.filename, transformation.Hash())
		})
	}
}
