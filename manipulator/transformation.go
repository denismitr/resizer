package manipulator

import (
	"github.com/pkg/errors"
)

var ErrBadTransformationRequest = errors.New("manipulator bad transformation request")

const (
	Rotate90 Degrees = 90
	Rotate180 Degrees = 180
	Rotate270 Degrees = 270
)

const (
	JPEG Format = "jpeg"
	TIFF Format = "tiff"
	WEBP Format = "webp"
)

type Percent uint16
type Degrees int64
type Pixels uint64
type Format string

type Flip struct {
	Horizontal bool
	Vertical   bool
}

type Transformation struct {
	Height     Pixels
	Width      Pixels
	Proportion Percent
	Quality    Percent
	Rotation   Degrees
	Format     Format
	Flip       Flip
}

func NewTransformation(format string, height, width, proportion, quality, rotation int) (Transformation, error) {
	var t Transformation
	if height < 0 || height > 0xFFFF {
		return t, errors.Wrapf(ErrBadTransformationRequest, "height value is invalid")
	}

	if width < 0 || width > 0xFFFF {
		return t, errors.Wrapf(ErrBadTransformationRequest, "width value is invalid")
	}

	// todo: validate others

	return Transformation{
		Format:     Format(format),
		Width:      Pixels(width),
		Height:     Pixels(height),
		Proportion: Percent(proportion),
		Quality:    Percent(quality),
		Rotation:   Degrees(rotation),
	}, nil
}

func (t Transformation) Empty() bool {
	return t.Height != 0 && t.Width != 0 // todo: others
}
