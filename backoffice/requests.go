package backoffice

import "io"

type createImageUseCase struct {
	name          string
	originalName  string
	originalExt   string
	publish       bool
	originalSize  int64
	namespace     string
	source        io.ReadSeeker
	originalSlice *createSliceUseCase
}

type createSliceUseCase struct {
	imageID   string
	filename  string
	path      string
	extension string
	namespace string
	size      int
	width     int
	height    int
}
