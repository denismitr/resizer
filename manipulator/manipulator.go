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

type Manipulator struct {
	scaleUp bool
}

func New() *Manipulator {
	return &Manipulator{}
}

func (m *Manipulator) Transform(source io.Reader, dst io.Writer, t *Transformation) error {
	if t == nil {
		panic("how can a transformation object be nil")
	}

	img, sourceFormat, err := image.Decode(source)
	if err != nil {
		return errors.Wrap(ErrBadImage, err.Error())
	}

	if originalFormatIsJpegOrGif(sourceFormat) {
		lr := io.LimitReader(source, maxExifSize)
		exifTransformation := computeExifOrientation(lr);
		if exifTransformation != nil && ! exifTransformation.None() {
			transformedImg, err := m.transform(img, exifTransformation);
			if err != nil {
				return err
			}

			img = transformedImg
		}
	}

	var targetFormat Format
	if sourceFormat == string(TIFF) || sourceFormat == string(WEBP) {
		targetFormat = JPEG
	}

	if t.Format != "" {
		targetFormat = t.Format
	}

	switch targetFormat {
	case JPEG:
		return m.transformJpeg(img, dst, t)
	case PNG:
		return m.transformPng(img, dst, t)
	default:
		panic(fmt.Sprintf("unsupported format %v", targetFormat))
	}

	return nil
}

func (m *Manipulator) transform(img image.Image, t *Transformation) (image.Image, error) {
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

func (m *Manipulator) resize(img image.Image, t *Transformation) (image.Image, error) {
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

		img = imaging.Resize(img, int(t.Resize.Height), int(t.Resize.Width), imaging.Lanczos)
	}

	return img, nil
}

func (m *Manipulator) outOfBoundaries(x, y int, resize Resize) bool {
	if !m.scaleUp && (int(resize.Height) > y) || (int(resize.Width) > x) {
		return true
	}

	return false
}

func (m *Manipulator) transformJpeg(img image.Image, dst io.Writer, t *Transformation) error {
	q := t.Quality
	if q == 0 {
		q = jpeg.DefaultQuality
	}

	transformedImg, err := m.transform(img, t);
	if err != nil {
		return err
	}

	if err := jpeg.Encode(dst, transformedImg, &jpeg.Options{Quality: int(q)}); err != nil {
		return errors.Wrapf(ErrTransformationFailed, "could not encode image to jpeg %v", err)
	}

	return nil
}

func (m *Manipulator) transformPng(img image.Image, dst io.Writer, t *Transformation) error {
	transformedImg, err := m.transform(img, t)
	if err != nil {
		return err
	}

	// todo: thing about quality
	if err := png.Encode(dst, transformedImg); err != nil {
		return errors.Wrapf(ErrTransformationFailed, "could not encode image to png %v", err)
	}

	return nil
}

//func (m *Manipulator) crop(img image.Image, t Transformation) image.Rectangle {
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

	exf, err := exif.Decode(r);
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