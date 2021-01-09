package manipulator

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"resizer/media"
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
			expected: 530,
		},
		{
			original: 530,
			desired:  123,
			upscale:  false,
			step:     20,
			expected: 130,
		},
		{
			original: 530,
			desired:  11,
			upscale:  false,
			step:     20,
			expected: 10,
		},
		{
			original: 90,
			desired:  27,
			upscale:  false,
			step:     20,
			expected: 30,
		},
		{
			original: 90,
			desired:  53,
			upscale:  false,
			step:     25,
			expected: 65,
		},
		{
			original: 530,
			desired:  10,
			upscale:  false,
			step:     20,
			expected: 10,
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
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%d of %d step %d upscale %v", tc.desired, tc.original, tc.step, tc.upscale),  func(t *testing.T) {
			p := Normalizer{cfg: &Config{
				AllowUpscale: tc.upscale,
				SizeDiscreteStep: tc.step,
			}}

			result := p.calculateNearestPixels(tc.original, tc.desired)

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
				Mime: "image/jpeg",
				Slug: "original",
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
				Mime: "image/png",
				Slug: "original",
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
				Mime: "image/png",
				Slug: "original",
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
				Mime: "image/png",
				Slug: "original",
			},
			filename: "h200_q80_s50_w400.png",
		},
	}

	m := New(&Config{})

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s-%s", tc.requestedTransformations, tc.extension), func(t *testing.T) {
			transformation, err := m.Convert(tc.requestedTransformations, tc.extension)
			defer m.Reset(transformation)
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
