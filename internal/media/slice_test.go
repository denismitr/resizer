package media

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComputeSlicePath(t *testing.T) {
	id := ID("foobar")
	namespace := "baz"
	filename := "image.png"

	result := ComputeSlicePath(namespace, id, filename)

	assert.Equal(t, "baz/foobar/image.png", result)
}

//func TestSlice_GetFileNameFromPath(t *testing.T) {
//	s := Slice{
//		Namespace: "foobar",
//		ImageID: ID("baz"),
//		Path: "foobar/bar",
//	}
//
//	result := s.GetFileNameFromPath()
//
//	assert.Equal(t, "", result)
//}
