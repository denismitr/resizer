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
			t.Resize.Height = Pixels(
				calculateNearestPixels(n.cfg.SizeDiscreteStep, originalHeight, int(t.Resize.Height), n.cfg.AllowUpscale),
			)
		}

		if t.Resize.Width != 0 {
			t.Resize.Width = Pixels(
				calculateNearestPixels(n.cfg.SizeDiscreteStep, originalWidth, int(t.Resize.Width), n.cfg.AllowUpscale),
			)
		}

		if t.Resize.Scale != 0 && t.Resize.Scale != 100 {
			t.Resize.Scale = Percent(
				calculatePercent(n.cfg.ScaleDiscreteStep, 100, int(t.Resize.Scale), n.cfg.AllowUpscale),
			)
		}
	}

	if t.Quality != 0 && t.Quality != 100 {
		t.Quality = Percent(
			calculatePercent(n.cfg.QualityDiscreteStep, 100, int(t.Quality), n.cfg.AllowUpscale),
		)
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

func calculateNearestPixels(step, originalPixels, desiredPixels int, upscale bool) int {
	if originalPixels < 0 || desiredPixels < 0 {
		panic("how could the validation system let pixels be less than zero")
	}

	if desiredPixels == 0 {
		return 0
	}

	if step == 0 || originalPixels == 0 {
		return desiredPixels
	}

	if !upscale && desiredPixels > originalPixels {
		return originalPixels
	}

	if desiredPixels < step {
		return step
	}

	var nearest int
	remainder := desiredPixels % step
	if remainder > (step / 2) {
		nearest = desiredPixels - remainder + step
	} else if (originalPixels - desiredPixels) < remainder {
		nearest = originalPixels
	} else {
		nearest = desiredPixels - remainder
	}

	return closest(nearest, originalPixels, upscale)
}

func calculatePercent(step, originalPercent, desiredPercent int, upscale bool) int {
	if originalPercent < 0 || desiredPercent < 0 {
		panic("how can percents be less than zero")
	}

	if desiredPercent < minPercent || desiredPercent == maxPercent {
		return 0
	}

	if !upscale && desiredPercent > originalPercent {
		return originalPercent
	}

	if step == 0 || originalPercent == 0  {
		return desiredPercent
	}

	if desiredPercent < step {
		return step
	}

	var nearest int
	remainder := desiredPercent % step
	if remainder > (step / 2) {
		nearest = desiredPercent - remainder + step
	} else {
		nearest = desiredPercent - remainder
	}

	return closest(nearest, originalPercent, upscale)
}

func closest(nearest, original int, upscale bool) int {
	if upscale {
		return nearest
	}

	return min(nearest, original)
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}