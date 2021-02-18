package backoffice

import (
	"github.com/denismitr/resizer/internal/media"
	"io"
)

type createImageDTO struct {
	name          string
	originalName  string
	originalExt   string
	publish       bool
	originalSize  int64
	namespace     string
	source        io.ReadSeeker
}

type originalSlice struct {
	slice *media.Slice
	content io.Reader
}
