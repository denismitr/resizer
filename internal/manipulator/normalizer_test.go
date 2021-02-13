package manipulator

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/denismitr/resizer/internal/media"
	"testing"
)

func Test_calculateNearestPixels(t *testing.T) {
	tt := []struct{
		original int
		desired  int
		upscale  bool
		step     int
		expected int
	}{
		{
			original: 530,
			desired:  523,
			upscale:  false,
			step:     20,
			expected: 520,
		},
		{
			original: 530,
			desired:  1,
			upscale:  false,
			step:     20,
			expected: 20,
		},
		{
			original: 530,
			desired:  123,
			upscale:  false,
			step:     25,
			expected: 125,
		},
		{
			original: 530,
			desired:  11,
			upscale:  false,
			step:     20,
			expected: 20,
		},
		{
			original: 90,
			desired:  27,
			upscale:  false,
			step:     15,
			expected: 30,
		},
		{
			original: 90,
			desired:  53,
			upscale:  false,
			step:     26,
			expected: 52,
		},
		{
			original: 530,
			desired:  10,
			upscale:  false,
			step:     20,
			expected: 20,
		},
		{
			original: 530,
			desired:  0,
			upscale:  false,
			step:     20,
			expected: 0,
		},
		{
			original: 530,
			desired:  700,
			upscale:  false,
			step:     20,
			expected: 530,
		},
		{
			original: 530,
			desired:  527,
			upscale:  false,
			step:     20,
			expected: 530,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%d of %d step %d upscale %v", tc.desired, tc.original, tc.step, tc.upscale),  func(t *testing.T) {
			result := calculateNearestPixels(tc.step, tc.original, tc.desired, tc.upscale)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_calculateNearestPercent(t *testing.T) {
	tt := []struct{
		original int
		desired  int
		upscale  bool
		step     int
		expected int
	}{
		{
			original: 100,
			desired:  90,
			upscale:  false,
			step:     10,
			expected: 90,
		},
		{
			original: 100,
			desired:  1,
			upscale:  false,
			step:     20,
			expected: 20,
		},
		{
			original: 100,
			desired:  99,
			upscale:  false,
			step:     15,
			expected: 100,
		},
		{
			original: 75,
			desired:  99,
			upscale:  false,
			step:     15,
			expected: 75,
		},
		{
			original: 75,
			desired:  44,
			upscale:  false,
			step:     20,
			expected: 40,
		},
		{
			original: 100,
			desired:  44,
			upscale:  false,
			step:     18,
			expected: 36,
		},
		{
			original: 100,
			desired:  52,
			upscale:  false,
			step:     18,
			expected: 54,
		},
		{
			original: 100,
			desired:  109,
			upscale:  true,
			step:     15,
			expected: 105,
		},
		{
			original: 100,
			desired:  113,
			upscale:  true,
			step:     15,
			expected: 120,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%d of %d step %d upscale %v", tc.desired, tc.original, tc.step, tc.upscale),  func(t *testing.T) {
			result := calculatePercent(tc.step, tc.original, tc.desired, tc.upscale)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateTransformation_DefaultCfg(t *testing.T) {
	tt := []struct{
		requestedTransformations string
		extension string
		img *media.Image
		err error
		expected *Transformation
		filename string
	}{
		{
			requestedTransformations: "h200",
			extension: "jpg",
			img: &media.Image{
				OriginalSlice: &media.Slice{Width: 500, Height: 400},
			},
			expected: &Transformation{
				Mime:      "image/jpeg",
				Resize:    Resize{Height: 200, Width: 0},
				Extension: JPEG,
			},
			filename: "h200.jpg",
		},
		{
			requestedTransformations: "h200_w400",
			extension: "png",
			img: &media.Image{
				OriginalSlice: &media.Slice{Width: 500, Height: 300},
			},
			expected: &Transformation{
				Resize:    Resize{Height: 200, Width: 400},
				Extension: PNG,
				Mime:      "image/png",
			},
			filename: "h200_w400.png",
		},
		{
			requestedTransformations: "h200_w400_q80",
			extension: "png",
			img: &media.Image{
				OriginalSlice: &media.Slice{Width: 500, Height: 900},
			},
			expected: &Transformation{
				Resize:    Resize{Height: 200, Width: 400},
				Quality:   80,
				Extension: PNG,
				Mime:      "image/png",
			},
			filename: "h200_q80_w400.png",
		},
		{
			requestedTransformations: "h200_w400_q80_s50",
			extension: "png",
			img: &media.Image{
				OriginalSlice: &media.Slice{Width: 500},
			},
			expected: &Transformation{
				Resize:    Resize{Height: 200, Width: 400, Scale: 50},
				Quality:   80,
				Extension: PNG,
				Mime:      "image/png",
			},
			filename: "h200_q80_s50_w400.png",
		},
	}

	m := New(&Config{})

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s-%s", tc.requestedTransformations, tc.extension), func(t *testing.T) {
			transformation, err := m.Convert(tc.requestedTransformations, tc.extension)
			if !assert.NoError(t, err) {
				t.Fatal(err.(*ValidationError).Errors())
			}

			err = m.Normalize(transformation, tc.img)
			if tc.err != nil {
				assert.Error(t, err)
				return
			}

			if ! assert.NoError(t, err) {
				t.Fatal(err)
			}

			assert.NotNil(t, transformation)
			assert.Equal(t, tc.expected, transformation)
			assert.Equal(t, tc.filename, transformation.Filename())
		})
	}
}
