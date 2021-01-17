package manipulator

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
)

type prefix string

const (
	height         prefix = "h"
	width          prefix = "w"
	scale          prefix = "s"
	quality        prefix = "q"
	opacity        prefix = "o"
	crop           prefix = "c"
	cropLeft       prefix = "cl"
	cropRight      prefix = "cr"
	cropTop        prefix = "cr"
	cropBottom     prefix = "cr"
	flipHorizontal prefix = "fh"
	flipVertical   prefix = "fv"
	fit            prefix = "fit"
	rotate90       prefix = "r90"
	rotate180      prefix = "r180"
	rotate270      prefix = "r270"
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
	maxPercent = 100
	minPercent = 1
)

var mimes = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
}

type integerCheck struct {
	name         string
	rx           *regexp.Regexp
	min          int
	max          int
	defaultValue int
	segment      prefix
	setter       func(v int, t *Transformation)
}

type booleanCheck struct {
	name         string
	segment      prefix
	defaultValue bool
	setter       func(hasMatch bool, t *Transformation)
}

type paramConverter struct {
	intChecks       []integerCheck
	validExtensions []string
	cfg             *Config
	boolChecks      []booleanCheck
}

func (pc *paramConverter) convertTo(
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

	if !pc.isValidExtension(requestedExtension) {
		vErr.Add("extension", fmt.Sprintf("unsupported extension %s", requestedExtension))
		return vErr
	}

SegmentsLoop:
	for _, s := range segments {
		for _, check := range pc.intChecks {
			v, err := matchInteger(check.rx, s, check.min, check.max)
			if err != nil {
				vErr.Add(check.name, err.Error())
				break SegmentsLoop
			} else if v != 0 && v != check.defaultValue {
				check.setter(v, t)
				continue SegmentsLoop
			}
		}

		for _, check := range pc.boolChecks {
			if s == string(check.segment) {
				check.setter(true, t)
				continue SegmentsLoop
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

func newParamConverter(cfg *Config) *paramConverter {
	integerChecks := []integerCheck{
		{
			name:    "height",
			rx:      regexp.MustCompile(`^h(\d{1,5})$`),
			min:     minHeight,
			max:     maxHeight,
			segment: height,
			setter:  func(v int, t *Transformation) { t.Resize.Height = Pixels(v) },
		},
		{
			name:    "width",
			rx:      regexp.MustCompile(`^w(\d{1,5})$`),
			min:     minWidth,
			max:     maxWidth,
			segment: width,
			setter:  func(v int, t *Transformation) { t.Resize.Width = Pixels(v) },
		},
		{
			name:         "scale",
			rx:           regexp.MustCompile(`^s(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: maxPercent,
			segment:      scale,
			setter:       func(v int, t *Transformation) { t.Resize.Scale = Percent(v) },
		},
		{
			name:         "quality",
			rx:           regexp.MustCompile(`^q(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: maxPercent,
			segment:      quality,
			setter:       func(v int, t *Transformation) { t.Quality = Percent(v) },
		},
		{
			name:         "opacity",
			rx:           regexp.MustCompile(`^o(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      opacity,
			setter:       func(v int, t *Transformation) { t.Opacity = Percent(v) },
		},
		{
			name:         "crop",
			rx:           regexp.MustCompile(`^c(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      crop,
			setter: func(v int, t *Transformation) {
				t.Resize.Crop.Left = Percent(v)
				t.Resize.Crop.Top = Percent(v)
				t.Resize.Crop.Right = Percent(v)
				t.Resize.Crop.Bottom = Percent(v)
			},
		},
		{
			name:         "cropLeft",
			rx:           regexp.MustCompile(`^cl(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      cropLeft,
			setter: func(v int, t *Transformation) {
				t.Resize.Crop.Left = Percent(v)
			},
		},
		{
			name:         "cropRight",
			rx:           regexp.MustCompile(`^cr(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      cropRight,
			setter: func(v int, t *Transformation) {
				t.Resize.Crop.Right = Percent(v)
			},
		},
		{
			name:         "cropTop",
			rx:           regexp.MustCompile(`^ct(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      cropTop,
			setter: func(v int, t *Transformation) {
				t.Resize.Crop.Top = Percent(v)
			},
		},
		{
			name:         "cropBottom",
			rx:           regexp.MustCompile(`^cb(\d{1,3})$`),
			min:          minPercent,
			max:          maxPercent,
			defaultValue: 0,
			segment:      cropBottom,
			setter: func(v int, t *Transformation) {
				t.Resize.Crop.Bottom = Percent(v)
			},
		},
	}

	boolChecks := []booleanCheck{
		{
			name:         "flipVertical",
			segment:      flipVertical,
			defaultValue: false,
			setter: func(hasMatch bool, t *Transformation) {
				t.Flip.Vertical = hasMatch
			},
		},
		{
			name:         "flipHorizontal",
			segment:      flipHorizontal,
			defaultValue: false,
			setter: func(hasMatch bool, t *Transformation) {
				t.Flip.Horizontal = hasMatch
			},
		},
		{
			name:         "fit",
			segment:      fit,
			defaultValue: false,
			setter: func(hasMatch bool, t *Transformation) {
				t.Resize.Fit = hasMatch
			},
		},
		{
			name:         "rotate90",
			defaultValue: false,
			segment:      rotate90,
			setter: func(hasMatch bool, t *Transformation) {
				if hasMatch {
					t.Rotation = Rotate90
				}
			},
		},
		{
			name:         "rotate180",
			defaultValue: false,
			segment:      rotate180,
			setter: func(hasMatch bool, t *Transformation) {
				if hasMatch {
					t.Rotation = Rotate180
				}
			},
		},
		{
			name:         "rotate270",
			defaultValue: false,
			segment:      rotate270,
			setter: func(hasMatch bool, t *Transformation) {
				if hasMatch {
					t.Rotation = Rotate270
				}
			},
		},
	}

	return &paramConverter{
		intChecks:       integerChecks,
		boolChecks:      boolChecks,
		validExtensions: []string{"jpg", "jpeg", "png"},
		cfg:             cfg,
	}
}

func matchInteger(rx *regexp.Regexp, input string, min, max int) (int, error) {
	match := rx.FindStringSubmatch(input)
	if len(match) > 1 {
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

func (pc *paramConverter) isValidExtension(ext string) bool {
	for _, vExt := range pc.validExtensions {
		if ext == vExt {
			return true
		}
	}

	return false
}
