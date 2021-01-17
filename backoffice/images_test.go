package backoffice

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServer_createUrlFriendlyName(t *testing.T) {
	tt := []struct{
		name string
		originalExt string
		originalName string
		expected string
	}{
		{name: "", originalExt: "jpg", originalName: "foo bar baz.jpg", expected: "foo-bar-baz.jpg"},
		{name: "", originalExt: "jpg", originalName: "foo bar baz.jpg", expected: "foo-bar-baz.jpg"},
		{name: "", originalExt: "png", originalName: "Screenshot 2020-12-17 at 14.07.18.png", expected: "screenshot-2020-12-17-at-14-07-18.png"},
		{name: "foo_bar", originalExt: "png", originalName: "foo bar baz.jpg", expected: "foo_bar.png"},
	}

	for i, tc := range tt {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			result := createURLFriendlyName(&createImageUseCase{
				name: tc.name,
				originalExt: tc.originalExt,
				originalName: tc.originalName,
			})

			assert.Equal(t, tc.expected, result)
		})
	}
}
