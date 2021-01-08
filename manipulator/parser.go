package manipulator

import (
	"fmt"
	"resizer/media"
)

type ValidationError struct {
	errors map[string]string
}

func NewValidationError() *ValidationError {
	return &ValidationError{errors: make(map[string]string)}
}

func (err *ValidationError) Add(k, v string) {
	err.errors[k] = v
}

func (err *ValidationError) Empty() bool {
	return len(err.errors) == 0
}

func (err *ValidationError) Error() string {
	return fmt.Sprint("Validation")
}

func (err *ValidationError) Errors() map[string]string {
	return err.errors
}

type Parser struct {
	tokenizer Tokenizer
	cfg       *Config
}

func NewParser(cfg *Config) *Parser {
	return &Parser{
		cfg: cfg,
		tokenizer: NewRegexTokenizer(cfg),
	}
}

func (p *Parser) Tokenize(requestedTransformations, requestedExtension string) (*Parameters, error) {
	return p.tokenizer.Tokenize(requestedTransformations, requestedExtension)
}

func (p *Parser) Parse(img *media.Image, parameters *Parameters) (*Transformation, error) {
	t := Transformation{
		Extension: Extension(parameters.extension),
	}

	originalWidth := img.OriginalSlice.Width
	originalHeight := img.OriginalSlice.Height

	if parameters.WantsTransformation() {
		if parameters.HasHeight() {
			t.Resize.Height = Pixels(p.calculateNearestPixels(originalHeight, parameters.Height()))
		}

		if parameters.HasWidth() {
			t.Resize.Width = Pixels(p.calculateNearestPixels(originalWidth, parameters.Width()))
		}

		if parameters.HasScale() {
			t.Resize.Scale = Percent(parameters.Scale())
		}
	}

	if parameters.HasQuality() {
		t.Quality = Percent(parameters.Quality())
	}

	mime := parameters.MimeType()

	switch mime {
	case "image/png":
		t.Extension = PNG
	case "image/jpeg":
		t.Extension = JPEG
	default:
		return nil, &ValidationError{
			errors: map[string]string{"format": fmt.Sprintf("Extension %s is unsupported", mime)},
		}
	}

	return &t, nil
}

func (p *Parser) calculateNearestPixels(originalPixels, desiredPixels int) int {
	if originalPixels < 0 || desiredPixels < 0 {
		panic("how can pixels be less than zero")
	}

	if desiredPixels == 0 {
		return 0
	}

	if p.cfg.SizeDiscreteStep == 0 || originalPixels == 0 {
		return desiredPixels
	}

	if !p.cfg.AllowUpscale && desiredPixels > originalPixels {
		return originalPixels
	}

	nearest := ((originalPixels - desiredPixels) % p.cfg.SizeDiscreteStep) + desiredPixels

	return closest(desiredPixels, p.cfg.SizeDiscreteStep, nearest)
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