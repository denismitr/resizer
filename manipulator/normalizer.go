package manipulator

import (
	"fmt"
	"resizer/media"
)

type normalizer struct {
	cfg *Config
}

func newNormalizer(cfg *Config) *normalizer {
	return &normalizer{cfg: cfg}
}

func (n *normalizer) normalize(t *Transformation, img *media.Image) error {
	originalWidth := img.OriginalSlice.Width
	originalHeight := img.OriginalSlice.Height

	if t.RequiresResize() {
		if t.Resize.Height != 0 {
			t.Resize.Height = Pixels(n.calculateNearestPixels(originalHeight, int(t.Resize.Height)))
		}

		if t.Resize.Width != 0 {
			t.Resize.Width = Pixels(n.calculateNearestPixels(originalWidth, int(t.Resize.Width)))
		}

		if t.Resize.Scale != 0 && t.Resize.Scale != 100 {
			t.Resize.Scale = Percent(n.calculatePercent(100, int(t.Resize.Scale)))
		}
	}

	if t.Quality != 0 && t.Quality != 100 {
		t.Quality = Percent(n.calculatePercent(100, int(t.Quality)))
	}

	switch t.Mime {
	case "image/png":
		t.Extension = PNG
	case "image/jpeg":
		t.Extension = JPEG
	default:
		return &ValidationError{
			errors: map[string]string{"format": fmt.Sprintf("Extension %s is unsupported", t.Mime)},
		}
	}

	return nil
}

func (n *normalizer) calculateNearestPixels(originalPixels, desiredPixels int) int {
	if originalPixels < 0 || desiredPixels < 0 {
		panic("how can pixels be less than zero")
	}

	if desiredPixels == 0 {
		return 0
	}

	if n.cfg.SizeDiscreteStep == 0 || originalPixels == 0 {
		return desiredPixels
	}

	if !n.cfg.AllowUpscale && desiredPixels > originalPixels {
		return originalPixels
	}

	nearest := ((originalPixels - desiredPixels) % n.cfg.SizeDiscreteStep) + desiredPixels

	return closest(desiredPixels, n.cfg.SizeDiscreteStep, nearest)
}

func (n *normalizer) calculatePercent(originalPercent, desiredPercent int) int {
	if originalPercent < 0 || desiredPercent < 0 || desiredPercent > 100 {
		panic("how can percents be less than zero or greater than 100")
	}

	if desiredPercent == 0 || desiredPercent == 100 {
		return 0
	}

	if n.cfg.QualityDiscreteStep == 0 || originalPercent == 0  {
		return desiredPercent
	}

	nearest := ((originalPercent - desiredPercent) % n.cfg.QualityDiscreteStep) + desiredPercent

	return closest(desiredPercent, n.cfg.SizeDiscreteStep, nearest)
}

func closest(desired, step, nearest int) int {
	a := nearest - step
	b := nearest - desired
	if a > 0 && a < b {
		return a
	} else {
		return nearest
	}
}
