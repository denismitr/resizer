package backoffice

import "io"

type createNewImage struct {
	name          string
	originalName  string
	originalExt   string
	originalSize  int64
	bucket        string
	source        io.ReadSeeker
	originalSlice *createNewSlice
}

type createNewSlice struct {
	originalID string
	filename   string
	path   string
	extension  string
	bucket     string
	size       int
	width      int
	height     int
}
