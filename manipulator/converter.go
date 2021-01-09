package manipulator

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
)

type Prefix string

const (
	HeightPrefix  Prefix = "h"
	WidthPrefix   Prefix = "w"
	ScalePrefix   Prefix = "s"
	QualityPrefix Prefix = "q"
	OpacityPrefix Prefix = "o"
)

const (
	minHeight  = 1
	maxHeight  = 10000
	minWidth   = 1
	maxWidth   = 10000
	minQuality = 1
	maxQuality = 100
	minScale   = 1
	maxScale   = 100
)

var mimes = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
}

type integerCheck struct {
	name string
	rx *regexp.Regexp
	min int
	max int
	defaultValue int
	segment Prefix
	setter func(v int, t *Transformation)
}

type ParamConverter struct {
	intChecks       []integerCheck
	validExtensions []string
	cfg             *Config
}

func (pc *ParamConverter) ConvertTo(
	t *Transformation,
	requestedTransformations,
	requestedExtension string,
) error {
	segments := strings.Split(strings.Trim(requestedTransformations, "/ "), "_")

	vErr := NewValidationError()
	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		vErr.Add("segments", "no segments provided")
		return vErr
	}

	if ! pc.isValidExtension(requestedExtension) {
		vErr.Add("extension", fmt.Sprintf("unsupported extension %s", requestedExtension))
		return vErr
	}

	for _, s := range segments {
		for _, check := range pc.intChecks {
			v, err := matchInteger(check.rx, s, check.min, check.max)
			if err != nil {
				vErr.Add(check.name, err.Error())
				continue
			} else if v != 0 && v != check.defaultValue {
				check.setter(v, t)
				continue
			}
		}
	}

	if t.Empty() {
		vErr.Add("segments", "no valid segments provided")
		return vErr
	}

	if !vErr.Empty() {
		return vErr
	}

	if requestedExtension == "jpeg" {
		t.Extension = "jpg"
		t.Mime = "image/jpeg"
	} else {
		t.Extension = Extension(requestedExtension)
		if m, ok := mimes[requestedExtension]; ok {
			t.Mime = m
		} else {
			panic("Mime " + requestedExtension)
		}
	}

	return nil
}

func NewRegexParamConverter(cfg *Config) *ParamConverter {
	checks := []integerCheck{
		{
			name: "height",
			rx: regexp.MustCompile(`^h(\d{1,5})$`),
			min: minHeight,
			max: maxHeight,
			segment: HeightPrefix,
			setter: func(v int, t *Transformation) { t.Resize.Height = Pixels(v) },
		},
		{
			name: "width",
			rx: regexp.MustCompile(`^w(\d{1,5})$`),
			min: minWidth,
			max: maxWidth,
			segment: WidthPrefix,
			setter: func(v int, t *Transformation) { t.Resize.Width = Pixels(v) },
		},
		{
			name:         "scale",
			rx:           regexp.MustCompile(`^s(\d{1,3})$`),
			min:          minScale,
			max:          maxScale,
			defaultValue: maxScale,
			segment:      ScalePrefix,
			setter: func(v int, t *Transformation) { t.Resize.Scale = Percent(v) },
		},
		{
			name: "quality",
			rx: regexp.MustCompile(`^q(\d{1,3})$`),
			min: minQuality,
			max: maxQuality,
			defaultValue: maxQuality,
			segment: QualityPrefix,
			setter: func(v int, t *Transformation) { t.Quality = Percent(v) },
		},
		{
			name: "opacity",
			rx: regexp.MustCompile(`^o(\d{1,3})$`),
			min: 1,
			max: 100,
			defaultValue: 100,
			segment: OpacityPrefix,
			setter: func(v int, t *Transformation) { t.Opacity = Percent(v) },
		},
	}

	return &ParamConverter{
		intChecks:       checks,
		validExtensions: []string{"jpg", "jpeg", "png"},
		cfg:             cfg,
	}
}

func matchInteger(rx *regexp.Regexp, input string, min, max int) (int, error) {
	var match []string
	match = rx.FindStringSubmatch(input)
	if match != nil && len(match) > 1 {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, errors.Wrapf(ErrBadTransformationRequest, "invalid value %s", err.Error())
		}

		if value < min || value > max {
			return 0, errors.Wrapf(ErrBadTransformationRequest, "int value of %s must be between %d and %d", input, min, max)
		}

		return value, nil
	}

	return 0, nil
}

func (pc *ParamConverter) isValidExtension(ext string) bool {
	for _, vExt := range pc.validExtensions {
		if ext == vExt {
			return true
		}
	}

	return false
}
