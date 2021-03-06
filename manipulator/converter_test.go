package manipulator

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegexpConverter(t *testing.T) {
	type expected struct {
		mime string
		filename string
		height Pixels
		width Pixels
	}

	tt := []struct{
		requestedTransformations string
		extension                string
		err                      bool
		expected expected
	}{
		{
			requestedTransformations: "h200",
			extension: "jpg",
			expected: expected{
				filename: "h200.jpg",
				mime: "image/jpeg",
				height:   200,
				width:    0,
			},
		},
		{
			requestedTransformations: "h200",
			extension: "jpeg",
			expected: expected{
				filename: "h200.jpg",
				mime: "image/jpeg",
				height:   200,
				width:    0,
			},
		},
		{
			requestedTransformations: "h200_w400",
			extension: "png",
			expected: expected{
				filename: "h200_w400.png",
				mime: "image/png",
				height:   200,
				width:    400,
			},
		},
		{
			requestedTransformations: "h200_w400_fit",
			extension: "png",
			expected: expected{
				filename: "fit_h200_w400.png",
				mime: "image/png",
				height:   200,
				width:    400,
			},
		},
		{
			requestedTransformations: "r90_fit",
			extension: "jpg",
			expected: expected{
				filename: "r90.jpg",
				mime: "image/jpeg",
			},
		},
		{
			requestedTransformations: "r180_h200",
			extension: "jpg",
			expected: expected{
				filename: "h200_r180.jpg",
				mime: "image/jpeg",
				height: 200,
			},
		},
		{
			requestedTransformations: "r270_h210_w80",
			extension: "jpg",
			expected: expected{
				filename: "h210_r270_w80.jpg",
				mime: "image/jpeg",
				height: 210,
				width: 80,
			},
		},
		{
			requestedTransformations: "h200",
			extension: "foo", // invalid extension
			err: true,
			expected: expected{},
		},
		{
			requestedTransformations: "", // no transformation
			extension: "png",
			err: true,
			expected: expected{},
		},
		{
			requestedTransformations: "wxpo", // completely invalid transformations
			extension: "png",
			err: true,
			expected: expected{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.requestedTransformations, func(t *testing.T) {
			converter := newParamConverter(&Config{})
			transformation := new(Transformation)
			err := converter.convertTo(transformation, tc.requestedTransformations, tc.extension)
			if !tc.err && ! assert.NoError(t, err) {
				t.Fatalf("Fail: %v", err.(*ValidationError).Errors())
			} else if tc.err && !assert.Error(t, err) {
				t.Fatal("expected to see an error here")
			} else if tc.err {
				return
			}

			assert.Equal(t, tc.expected.height, transformation.Resize.Height)
			assert.Equal(t, tc.expected.width, transformation.Resize.Width)
			assert.Equal(t, tc.expected.filename, transformation.Filename())
			assert.Equal(t, tc.expected.mime, transformation.Mime)
		})
	}
}
