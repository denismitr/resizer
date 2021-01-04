package backoffice

import "io"

type createNewImage struct {
	name         string
	hash         string
	originalName string
	originalExt  string
	originalSize int64
	bucket       string
	source       io.ReadSeeker
	path         string
	url          string
	format       string
	size         int
	width        int
	height       int
}
