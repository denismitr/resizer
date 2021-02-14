package manipulator

import (
	"bytes"
	"fmt"
	"github.com/denismitr/resizer/internal/media"
	"github.com/disintegration/imaging"
	"github.com/pkg/errors"
	"github.com/rwcarlsen/goexif/exif"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"time"
)

type imageTransformer struct {
	cfg *Config
}

func newImageTransformer(cfg *Config) *imageTransformer {
	return &imageTransformer{
		cfg: cfg,
	}
}

func (it *imageTransformer) makeOriginalSlice(
	sourceImg image.Image,
	dst io.Writer,
	rootImage *media.Image,
	sourceFormat string,
) (*media.Slice, error) {
	originalTransformation, err := createOriginalTransformation(sourceImg, sourceFormat)
	if err != nil {
		return nil, err
	}

	return it.transform(sourceFormat, sourceImg, dst, rootImage, originalTransformation)
}

func (it *imageTransformer) decode(source io.Reader) (image.Image, string, error) {
	img, sourceFormat, err := image.Decode(source)
	if err != nil {
		return nil, "", errors.Wrap(ErrBadImage, err.Error())
	}

	if originalFormatIsJpegOrGif(sourceFormat) {
		lr := io.LimitReader(source, maxExifSize)
		exifTransformation := computeExifOrientation(lr)
		if exifTransformation != nil && !exifTransformation.None() {
			transformedImg, err := it.applyTransformationOn(img, exifTransformation)
			if err != nil {
				return nil, "", err
			}

			img = transformedImg
		}
	}

	return img, sourceFormat, nil
}

func (it *imageTransformer) createOriginalSlice(source io.Reader, dst io.Writer, rootImage *media.Image) (*media.Slice, error) {
	sourceImg, sourceFormat, err := it.decode(source)
	if err != nil {
		return nil, err
	}

	slice, err := it.makeOriginalSlice(sourceImg, dst, rootImage, sourceFormat)
	if err != nil {
		return nil, err
	}

	slice.IsValid = true
	slice.IsOriginal = true

	return slice, nil
}

func (it *imageTransformer) createSlice(source io.Reader, dst io.Writer, rootImage *media.Image, t *Transformation) (*media.Slice, error) {
	img, sourceFormat, err := it.decode(source)
	if err != nil {
		return nil, err
	}

	return it.transform(sourceFormat, img, dst, rootImage, t)
}

func (it *imageTransformer) transform(
	sourceExtension string,
	sourceImg image.Image,
	dst io.Writer,
	rootImage *media.Image,
	t *Transformation,
) (*media.Slice, error) {
	var targetFormat media.Extension
	if sourceExtension == "tiff" || sourceExtension == "webp" || sourceExtension == "jpg" || sourceExtension == "jpeg" {
		targetFormat = media.JPEG
	} else {
		targetFormat = media.PNG
	}

	if t.Extension != "" {
		targetFormat = t.Extension
	}

	buf := &bytes.Buffer{}
	createdImg, err := it.createTransformedImg(targetFormat, sourceImg, buf, t)
	if err != nil {
		return nil, err
	}

	n, err := io.Copy(dst, buf);
	if err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not copy bytes to dst; %v", err)
	}

	slice, err := rootImage.CreateSlice(
		targetFormat, t.Filename(),
		createdImg.Bounds().Dy(),
		createdImg.Bounds().Dx(),
		int(n),
		int(t.Quality),
		t.RequiresResize() && t.Resize.RequiresCrop(),
		time.Now(),
	)

	if err != nil {
		return nil, err
	}

	return slice, nil
}

func (it *imageTransformer) createTransformedImg(
	targetExtension media.Extension,
	original image.Image,
	dst io.Writer,
	t *Transformation,
) (image.Image, error) {
	switch targetExtension {
	case media.JPEG:
		return it.createJpeg(original, dst, t)
	case media.PNG:
		return it.createPng(original, dst, t)
	default:
		panic(fmt.Sprintf("unsupported format %v", targetExtension))
	}
}

func (it *imageTransformer) applyTransformationOn(img image.Image, t *Transformation) (image.Image, error) {
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

func (it *imageTransformer) resize(img image.Image, t *Transformation) (image.Image, error) {
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
		// on proportional resize we calculate height and width automatically
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

		if t.Resize.Fit && t.Resize.Width != 0 && t.Resize.Height != 0 {
			img = imaging.Fit(img, int(t.Resize.Width), int(t.Resize.Height), imaging.Lanczos)
		} else {
			img = imaging.Resize(img, int(t.Resize.Width), int(t.Resize.Height), imaging.Lanczos)
		}
	}

	return img, nil
}

func (it *imageTransformer) outOfBoundaries(x, y int, resize Resize) bool {
	if !it.cfg.AllowUpscale && (int(resize.Height) > y) || (int(resize.Width) > x) {
		return true
	}

	return false
}

func (it *imageTransformer) createJpeg(
	img image.Image,
	dst io.Writer,
	t *Transformation,
) (image.Image, error) {
	if t.Quality == 0 {
		t.Quality = 100
	}

	transformedImg, err := it.applyTransformationOn(img, t)
	if err != nil {
		return nil, err
	}

	if err := jpeg.Encode(dst, transformedImg, &jpeg.Options{Quality: int(t.Quality)}); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not transform image to jpeg %v", err)
	}

	return transformedImg, nil
}

func (it *imageTransformer) createPng(
	img image.Image,
	dst io.Writer,
	t *Transformation,
) (image.Image, error) {
	transformedImg, err := it.applyTransformationOn(img, t)
	if err != nil {
		return nil, err
	}

	if err := png.Encode(dst, transformedImg); err != nil {
		return nil, errors.Wrapf(ErrTransformationFailed, "could not transform image to png %v", err)
	}

	return transformedImg, nil
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

func createOriginalTransformation(img image.Image, sourceFormat string) (*Transformation, error) {
	ext, err :=  media.NormalizeExtension(sourceFormat)
	if err != nil {
		return nil, err
	}

	return &Transformation{
		Resize: Resize{
			Width:  Pixels(img.Bounds().Dx()),
			Height: Pixels(img.Bounds().Dy()),
		},
		Extension: ext,
	}, nil
}
