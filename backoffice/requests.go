package backoffice

import "io"

type createImageUseCase struct {
	name          string
	originalName  string
	originalExt   string
	publish       bool
	originalSize  int64
	bucket        string
	source        io.ReadSeeker
	originalSlice *createSliceUseCase
}

type createSliceUseCase struct {
	imageID   string
	filename  string
	path      string
	extension string
	bucket    string
	size      int
	width     int
	height    int
}
