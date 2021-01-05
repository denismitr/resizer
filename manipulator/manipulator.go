package manipulator

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

var ErrTransformationFailed = errors.New("manipulator bad transformation request")
var ErrBadImage = errors.New("manipulator bad image provided")

// maximum distance into image to look for EXIF tags
const maxExifSize = 1 << 20

type Manipulator interface {
	Transform(source io.Reader, dst io.Writer, t *Transformation) (*Transformed, error)
}

type StdManipulator struct {
	scaleUp bool
}

type Transformed struct {
	Width     int
	Height    int
	Extension string
	Filename  string
}

func New(scaleUp bool) *StdManipulator {
	return &StdManipulator{
		scaleUp: scaleUp,
	}
}

func (m *StdManipulator) original(img image.Image, dst io.Writer, sourceFormat string) (*Transformed, error) {
	r := &Transformed{
		Height:    img.Bounds().Dy(),
		Width:     img.Bounds().Dx(),
		Extension: sourceFormat,
	}

	r.Filename = fmt.Sprintf("h%d_w%d.%s", r.Height, r.Width, r.Extension) // fixme
	return m.encode(sourceFormat, img, dst, &Transformation{})
}

func (m *StdManipulator) Transform(source io.Reader, dst io.Writer, t *Transformation) (*Transformed, error) {
	img, sourceFormat, err := image.Decode(source)
	if err != nil {
		return nil, errors.Wrap(ErrBadImage, err.Error())
	}

	if t == nil {
		return m.original(img, dst, sourceFormat)
	}

	if originalFormatIsJpegOrGif(sourceFormat) {
		lr := io.LimitReader(source, maxExifSize)
		exifTransformation := computeExifOrientation(lr)
		if exifTransformation != nil && !exifTransformation.None() {
			transformedImg, err := m.transform(img, exifTransformation)
			if err != nil {
				return nil, err
			}

			img = transformedImg
		}
	}

	return m.encode(sourceFormat, img, dst, t)
}

func (m *StdManipulator) encode(
	sourceFormat string,
	img image.Image,
	dst io.Writer,
	t *Transformation,
) (*Transformed, error) {
	var targetFormat Format
	if sourceFormat == string(TIFF) || sourceFormat == string(WEBP) {
		targetFormat = JPEG
	} else {
		targetFormat = PNG
	}

	if t.Format != "" {
		targetFormat = t.Format
	}

	if t.Quality == 0 {
		t.Quality = 100
	}

	switch targetFormat {
	case JPEG:
		return m.transformJpeg(img, dst, t)
	case PNG:
		return m.transformPng(img, dst, t)
	default:
		panic(fmt.Sprintf("unsupported format %v", targetFormat))
	}
}

func (m *StdManipulator) transform(img image.Image, t *Transformation) (image.Image, error) {
	// todo: metrics

	if t.RequiresResize() {
		var resizeErr error
		img, resizeErr = m.resize(img, t)
		if resizeErr != nil {
			return nil, resizeErr
		}
	}

	if t.Rotation == Rotate90 {
		img = imaging.Rotate90(img)
	}

	if t.Rotation == Rotate180 {
		img = imaging.Rotate180(img)
	}

	if t.Rotation == Rotate270 {
		img = imaging.Rotate270(img)
	}

	if t.Flip.Horizontal {
		img = imaging.FlipH(img)
	}

	if t.Flip.Vertical {
		img = imaging.FlipV(img)
	}

	return img, nil
}

func (m *StdManipulator) resize(img image.Image, t *Transformation) (image.Image, error) {
	originalHeight := img.Bounds().Dy()
	originalWidth := img.Bounds().Dx()

	if t.Resize.RequiresCrop() {
		var x0, y0 int
		var x1 = originalWidth
		var y1 = originalHeight

		if t.Resize.Crop.Left != 0 {
			x0 = calculateDimensionAsProportion(originalWidth, t.Resize.Crop.Left)
		}

		if t.Resize.Crop.Top != 0 {
			y0 = calculateDimensionAsProportion(originalHeight, t.Resize.Crop.Top)
		}

		if t.Resize.Crop.Right != 0 {
			x1 = originalWidth - calculateDimensionAsProportion(originalWidth, t.Resize.Crop.Right)
		}

		if t.Resize.Crop.Bottom != 0 {
			y1 = originalHeight - calculateDimensionAsProportion(originalHeight, t.Resize.Crop.Bottom)
		}

		img = imaging.Crop(img, image.Rect(x0, y0, x1, y1))
	}

	if t.Resize.Proportion != 0 {
		// on proportional resize we calculate height and width automatically
		newHeight := calculateDimensionAsProportion(originalHeight, t.Resize.Proportion)
		newWidth := calculateDimensionAsProportion(originalWidth, t.Resize.Proportion)
		return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos), nil // fixme: crop and image.Filter
	}

	if t.Resize.WidthOrHeightProvided() {
		if m.outOfBoundaries(originalWidth, originalHeight, t.Resize) {
			return nil, errors.Wrapf(
				ErrBadTransformationRequest,
				"scale up is disabled: max height is %d, max width os %d",
				originalHeight, originalWidth,
			)
		}

		img = imaging.Resize(img, int(t.Resize.Width), int(t.Resize.Height), imaging.Lanczos)
	}

	return img, nil
}

func (m *StdManipulator) outOfBoundaries(x, y int, resize Resize) bool {
	if !m.scaleUp && (int(resize.Height) > y) || (int(resize.Width) > x) {
		return true
	}

	return false
}

func (m *StdManipulator) transformJpeg(img image.Image, dst io.Writer, t *Transformation) (*Transformed, error) {
	q := t.Quality
	if q == 0 {
		q = 100
	}

	transformedImg, err := m.transform(img, t)
	if err != nil {
		return nil, err
	}

	if err := jpeg.Encode(dst, transformedImg, &jpeg.Options{Quality: int(q)}); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not encode image to jpeg %v", err)
	}

	r := &Transformed{
		Height:    transformedImg.Bounds().Dy(),
		Width:     transformedImg.Bounds().Dx(),
		Extension: string(JPEG),
	}

	r.Filename = fmt.Sprintf("h%d_w%d.%s", r.Height, r.Width, r.Extension)

	return r, nil
}

func (m *StdManipulator) transformPng(img image.Image, dst io.Writer, t *Transformation) (*Transformed, error) {
	transformedImg, err := m.transform(img, t)
	if err != nil {
		return nil, err
	}

	// todo: thing about quality
	if err := png.Encode(dst, transformedImg); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not encode image to png %v", err)
	}

	r := &Transformed{
		Height:    transformedImg.Bounds().Dy(),
		Width:     transformedImg.Bounds().Dx(),
		Extension: string(PNG),
	}

	r.Filename = fmt.Sprintf("h%d_w%d.%s", r.Height, r.Width, r.Extension)

	return r, nil
}

//func (m *StdManipulator) crop(img image.Image, t Transformation) image.Rectangle {
//	return img.Bounds()
//}

func computeExifOrientation(r io.Reader) *Transformation {
	// Exif Orientation Tag values
	// http://sylvana.net/jpegcrop/exif_orientation.html
	const (
		topLeftSide     = 1
		topRightSide    = 2
		bottomRightSide = 3
		bottomLeftSide  = 4
		leftSideTop     = 5
		rightSideTop    = 6
		rightSideBottom = 7
		leftSideBottom  = 8
	)

	exf, err := exif.Decode(r)
	if err != nil {
		return nil
	}

	tag, err := exf.Get(exif.Orientation)
	if err != nil {
		return nil
	}

	orient, err := tag.Int(0)
	if err != nil {
		return nil
	}

	var t Transformation
	switch orient {
	case topLeftSide:
		// skip
	case topRightSide:
		t.Flip.Horizontal = true
	case bottomLeftSide:
		t.Flip.Vertical = true
	case bottomRightSide:
		t.Rotation = 180
	case leftSideTop:
		t.Rotation = 90
		t.Flip.Vertical = true
	case rightSideBottom:
		t.Rotation = 90
		t.Flip.Horizontal = true
	case leftSideBottom:
		t.Rotation = 90
	case rightSideTop:
		t.Rotation = -90
	}

	return &t
}

func originalFormatIsJpegOrGif(f string) bool {
	return f == "jpeg" || f == "tiff"
}

func calculateDimensionAsProportion(original int, proportion Percent) int {
	return int(float64(original) * (float64(proportion) / 100))
}
