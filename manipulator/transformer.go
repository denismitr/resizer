package manipulator

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

type ImageTransformer struct {
	cfg *Config
}

func NewImageTransformer(cfg *Config) *ImageTransformer {
	return &ImageTransformer{
		cfg: cfg,
	}
}

func (it *ImageTransformer) original(img image.Image, dst io.Writer, sourceFormat string) (*Result, error) {
	return it.encode(sourceFormat, img, dst, &Transformation{})
}

func (it *ImageTransformer) Transform(source io.Reader, dst io.Writer, t *Transformation) (*Result, error) {
	img, sourceFormat, err := image.Decode(source)
	if err != nil {
		return nil, errors.Wrap(ErrBadImage, err.Error())
	}

	if t == nil {
		return it.original(img, dst, sourceFormat)
	}

	if originalFormatIsJpegOrGif(sourceFormat) {
		lr := io.LimitReader(source, maxExifSize)
		exifTransformation := computeExifOrientation(lr)
		if exifTransformation != nil && !exifTransformation.None() {
			transformedImg, err := it.transform(img, exifTransformation)
			if err != nil {
				return nil, err
			}

			img = transformedImg
		}
	}

	return it.encode(sourceFormat, img, dst, t)
}

func (it *ImageTransformer) encode(
	sourceExtension string,
	img image.Image,
	dst io.Writer,
	t *Transformation,
) (*Result, error) {
	var targetFormat Extension
	if sourceExtension == "tiff" || sourceExtension == "webp" || sourceExtension == "jpg" || sourceExtension == "jpeg" {
		targetFormat = JPEG
	} else {
		targetFormat = PNG
	}

	if t.Extension != "" {
		targetFormat = t.Extension
	}

	switch targetFormat {
	case JPEG:
		return it.transformJpeg(img, dst, t)
	case PNG:
		return it.transformPng(img, dst, t)
	default:
		panic(fmt.Sprintf("unsupported format %v", targetFormat))
	}
}

func (it *ImageTransformer) transform(img image.Image, t *Transformation) (image.Image, error) {
	// todo: metrics

	if t.RequiresResize() {
		var resizeErr error
		img, resizeErr = it.resize(img, t)
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

func (it *ImageTransformer) resize(img image.Image, t *Transformation) (image.Image, error) {
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

	if t.Resize.Scale != 0 {
		// on proportional resize we calculate HeightPrefix and WidthPrefix automatically
		newHeight := calculateDimensionAsProportion(originalHeight, t.Resize.Scale)
		newWidth := calculateDimensionAsProportion(originalWidth, t.Resize.Scale)
		return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos), nil // fixme: crop and image.Filter
	}

	if t.Resize.WidthOrHeightProvided() {
		if it.outOfBoundaries(originalWidth, originalHeight, t.Resize) {
			return nil, errors.Wrapf(
				ErrBadTransformationRequest,
				"ScaleUp is disabled: max height is %d, max width is %d",
				originalHeight, originalWidth,
			)
		}

		img = imaging.Resize(img, int(t.Resize.Width), int(t.Resize.Height), imaging.Lanczos)
	}

	return img, nil
}

func (it *ImageTransformer) outOfBoundaries(x, y int, resize Resize) bool {
	if !it.cfg.AllowUpscale && (int(resize.Height) > y) || (int(resize.Width) > x) {
		return true
	}

	return false
}

func (it *ImageTransformer) transformJpeg(img image.Image, dst io.Writer, t *Transformation) (*Result, error) {
	result := &Result{Extension: string(JPEG)}

	q := t.Quality
	if q == 0 {
		q = 100
	} else {
		result.Quality = q
	}

	transformedImg, err := it.transform(img, t)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, transformedImg, &jpeg.Options{Quality: int(q)}); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not encode image to jpeg %v", err)
	}

	if n, err := io.Copy(dst, buf); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not copy bytes to dst; %v", err)
	} else {
		result.Size = int(n)
		result.Height = transformedImg.Bounds().Dy()
		result.Width = transformedImg.Bounds().Dx()
	}

	return result, nil
}

func (it *ImageTransformer) transformPng(img image.Image, dst io.Writer, t *Transformation) (*Result, error) {
	transformedImg, err := it.transform(img, t)
	if err != nil {
		return nil, err
	}

	// todo: thing about QualityPrefix
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, transformedImg); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not encode image to png %v", err)
	}

	r := &Result{
		Height:    transformedImg.Bounds().Dy(),
		Width:     transformedImg.Bounds().Dx(),
		Extension: string(PNG),
	}

	if n, err := io.Copy(dst, buf); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not copy bytes to dst; %v", err)
	} else {
		r.Size = int(n)
	}

	return r, nil
}

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
