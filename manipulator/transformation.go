package manipulator

import (
	"fmt"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

var ErrBadTransformationRequest = errors.New("manipulator bad transformation request")

const (
	Rotate90  Degrees = 90
	Rotate180 Degrees = 180
	Rotate270 Degrees = 270
)

const (
	JPEG Extension = "jpg"
	PNG  Extension = "png"
	TIFF Extension = "tiff"
	WEBP Extension = "webp"
)

type Percent uint16
type Degrees int64
type Pixels uint16
type Extension string

type Flip struct {
	Horizontal bool
	Vertical   bool
}

func (f Flip) None() bool {
	return !f.Vertical && !f.Horizontal
}

type Resize struct {
	Height Pixels
	Width  Pixels
	Scale  Percent
	Crop   Crop
}

func (r Resize) RequiresCrop() bool {
	return ! r.Crop.None()
}

type Crop struct {
	ContextBased bool
	Left         Percent
	Right        Percent
	Top          Percent
	Bottom       Percent
}

func (c Crop) Required() bool {
	return c.Left != 0 || c.Right != 0 || c.Bottom != 0 || c.Top != 0
}

func (c Crop) AllSides() bool {
	return c.Required() && c.Left == c.Right && c.Right == c.Bottom && c.Bottom == c.Top
}

func (c Crop) None() bool {
	return c.Left == 0 && c.Right == 0 && c.Bottom == 0 && c.Top == 0
}

func (r Resize) None() bool {
	return r.Crop.None() && r.Scale == 0 && r.Width == 0 && r.Height == 0
}

func (r Resize) Required() bool {
	return !r.None()
}

func (r Resize) WidthOrHeightProvided() bool {
	return r.Width != 0 || r.Height != 0
}

type Transformation struct {
	Resize    Resize
	Quality   Percent
	Rotation  Degrees
	Extension Extension
	Flip      Flip
	Mime      string
	Opacity   Percent
}

func (t *Transformation) None() bool {
	return t.Resize.None() && t.Flip.None()
}

func (t *Transformation) RequiresResize() bool {
	return !t.Resize.None()
}

func (t *Transformation) Filename() string {
	if t.Extension == "" {
		panic("how can extension be empty?")
	}

	var segments []string
	if t.Resize.Height != 0 {
		segments = append(segments, fmt.Sprintf("%s%d", height, t.Resize.Height))
	}

	if t.Resize.Width != 0 {
		segments = append(segments, fmt.Sprintf("%s%d", width, t.Resize.Width))
	}

	if t.Resize.Scale != 0 {
		segments = append(segments, fmt.Sprintf("%s%d", scale, t.Resize.Scale))
	}

	if t.Quality != 0 {
		segments = append(segments, fmt.Sprintf("%s%d", quality, t.Quality))
	}

	if t.Resize.Crop.Required() && t.Resize.Crop.AllSides() {
		segments = append(segments, fmt.Sprintf("%s%d", crop, t.Resize.Crop.Left))
	} else if t.Resize.Crop.Required() {
		if t.Resize.Crop.Left != 0 {
			segments = append(segments, fmt.Sprintf("%s%d", cropLeft, t.Resize.Crop.Left))
		}

		if t.Resize.Crop.Right != 0 {
			segments = append(segments, fmt.Sprintf("%s%d", cropRight, t.Resize.Crop.Right))
		}

		if t.Resize.Crop.Bottom != 0 {
			segments = append(segments, fmt.Sprintf("%s%d", cropBottom, t.Resize.Crop.Bottom))
		}

		if t.Resize.Crop.Top != 0 {
			segments = append(segments, fmt.Sprintf("%s%d", cropTop, t.Resize.Crop.Top))
		}
	}

	if t.Flip.Horizontal {
		segments = append(segments, fmt.Sprintf("%s", flipHorizontal))
	}

	if t.Flip.Vertical {
		segments = append(segments, fmt.Sprintf("%s", flipVertical))
	}

	sort.Strings(segments)

	return strings.ToLower(strings.Join(segments, "_") + "." + string(t.Extension))
}

func (t *Transformation) Empty() bool {
	return ! t.RequiresResize() && t.Quality == 0 && t.Rotation == 0 && t.Opacity == 0
}

func (t *Transformation) Reset() {
	t.Resize.Height = 0
	t.Resize.Width = 0
	t.Resize.Scale = 0
	t.Resize.Crop.Bottom = 0
	t.Resize.Crop.Top = 0
	t.Resize.Crop.Left = 0
	t.Resize.Crop.Right = 0
	t.Flip.Vertical = false
	t.Flip.Horizontal = false
	t.Opacity = 0
	t.Rotation = 0
	t.Mime = ""
	t.Extension = ""
}
