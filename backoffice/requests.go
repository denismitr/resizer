package backoffice

import "io"

type createNewImage struct {
	originalName string
	originalExt string
	originalSize int64
	bucket       string
	source       io.Reader
}
