package manipulator

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/jpeg"
	"io"
)

var ErrTransformationFailed = errors.New("manipulator bad transformation request")
var ErrBadImage = errors.New("manipulator bad image provided")

// maximum distance into image to look for EXIF tags
const maxExifSize = 1 << 20

type Manipulator struct {

}

func (m *Manipulator) Transform(r io.Reader, w io.Writer, t Transformation) error {
	img, sourceFormat, err := image.Decode(r)
	if err != nil {
		return errors.Wrap(ErrBadImage, err.Error())
	}

	if isJpegOrGif(sourceFormat) {
		lr := io.LimitReader(r, maxExifSize)
		if exifTransformation := computeExifOrientation(lr); ! exifTransformation.Empty() {
			if transformedImg, err := m.transform(img, exifTransformation); err != nil {
				return err
			} else {
				img = transformedImg
			}
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
		return m.transformJpeg(img, w, t)
	default:
		panic(fmt.Sprintf("unsupported format %v", targetFormat))
	}

	return nil
}

func (m *Manipulator) transform(img image.Image, t Transformation) (image.Image, error) {
	// todo: metrics

	//rect := m.crop(img, t)

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

func (m *Manipulator) transformJpeg(img image.Image, w io.Writer, t Transformation) error {
	q := t.Quality
	if q == 0 {
		q = jpeg.DefaultQuality
	}

	transformedImg, err := m.transform(img, t);
	if err != nil {
		return err
	}

	if err := jpeg.Encode(w, transformedImg, &jpeg.Options{Quality: int(q)}); err != nil {
		return errors.Wrapf(ErrTransformationFailed, "could not encode image to jpeg %v", err)
	}

	return nil
}

//func (m *Manipulator) crop(img image.Image, t Transformation) image.Rectangle {
//	return img.Bounds()
//}

func computeExifOrientation(r io.Reader) Transformation {
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

	var t Transformation
	exf, err := exif.Decode(r);
	if err != nil {
		return t
	}

	tag, err := exf.Get(exif.Orientation)
	if err != nil {
		return t
	}

	orient, err := tag.Int(0)
	if err != nil {
		return t
	}

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

	return t
}

func isJpegOrGif(f string) bool {
	return f == "jpeg" || f == "tiff"
}
