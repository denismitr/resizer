package manipulator

import (
	"github.com/pkg/errors"
)

var ErrBadTransformationRequest = errors.New("manipulator bad transformation request")

const (
	Rotate90  Degrees = 90
	Rotate180 Degrees = 180
	Rotate270 Degrees = 270
)

const (
	JPEG Format = "jpeg"
	PNG  Format = "png"
	TIFF Format = "tiff"
	WEBP Format = "webp"
)

type Percent uint16
type Degrees int64
type Pixels uint16
type Format string

type Flip struct {
	Horizontal bool
	Vertical   bool
}

func (f Flip) None() bool {
	return !f.Vertical && !f.Horizontal
}

type Resize struct {
	Height     Pixels
	Width      Pixels
	Proportion Percent
}

func (r Resize) None() bool {
	return r.Proportion == 0 && r.Width == 0 && r.Height == 0
}

func (r Resize) OnlyWidthOrHeightProvided() bool {
	return (r.Height != 0 && r.Width == 0) || (r.Height == 0 && r.Width != 0)
}

type Transformation struct {
	Resize   Resize
	Quality  Percent
	Rotation Degrees
	Format   Format
	Flip     Flip
}

func NewTransformation(format string, height, width, proportion, quality, rotation int) (*Transformation, error) {
	if height < 0 || height > 0xFFFF {
		return nil, errors.Wrapf(ErrBadTransformationRequest, "height value is invalid")
	}

	if width < 0 || width > 0xFFFF {
		return nil, errors.Wrapf(ErrBadTransformationRequest, "width value is invalid")
	}

	// todo: validate others

	return &Transformation{
		Format: Format(format),
		Resize: Resize{
			Width:      Pixels(width),
			Height:     Pixels(height),
			Proportion: Percent(proportion),
		},
		Quality:  Percent(quality),
		Rotation: Degrees(rotation),
	}, nil
}

func (t *Transformation) None() bool {
	return t.Resize.None() && t.Flip.None()
}

func (t *Transformation) RequiresResize() bool {
	return !t.Resize.None()
}
