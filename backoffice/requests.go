package backoffice

import "io"

type createNewImage struct {
	name         string
	originalName string
	originalExt  string
	originalSize int64
	bucket       string
	source       io.ReadSeeker
}
