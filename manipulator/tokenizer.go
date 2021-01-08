package manipulator

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type prefix string

const (
	height  prefix = "h"
	width   prefix = "w"
	scale   prefix = "s"
	quality prefix = "q"
	opacity prefix = "o"
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

type Parameters struct {
	slug      string
	extension string
	filename  string
	segments  map[prefix]int
}

func (p *Parameters) HasHeight() bool {
	return p.segments[height] != 0
}

func (p *Parameters) HasWidth() bool {
	return p.segments[width] != 0
}

func (p *Parameters) HasQuality() bool {
	return p.segments[quality] != 0
}

func (p *Parameters) HasScale() bool {
	return p.segments[scale] != 0
}

func (p *Parameters) Empty() bool {
	return len(p.segments) == 0
}

func (p *Parameters) WantsTransformation() bool {
	return p.segments[height] > 0 ||
		p.segments[width] > 0 ||
		(p.segments[quality] != 0 && p.segments[quality] != maxQuality) ||
		(p.segments[scale] != 0 && p.segments[scale] != maxScale)
}

func (p *Parameters) Filename() string {
	if p.filename != "" {
		return p.filename
	}

	var filenameParts []string
	for prefix := range p.segments {
		if v, ok := p.segments[prefix]; ok {
			filenameParts = append(filenameParts, fmt.Sprintf("%s%d", string(prefix), v))
		}
	}

	sort.Strings(filenameParts)

	p.filename = strings.ToLower(strings.Join(filenameParts, "_") + "." + p.extension)
	return p.filename
}

func (p *Parameters) MimeType() string {
	switch p.extension {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	default:
		panic("Unsupported mime " + p.extension)
	}
}

func (p *Parameters) Height() int {
	return p.segments[height]
}

func (p *Parameters) Width() int {
	return p.segments[width]
}

func (p *Parameters) Scale() int {
	return p.segments[scale]
}

func (p *Parameters) Quality() int {
	return p.segments[quality]
}

type Tokenizer interface {
	Tokenize(requestedTransformations, requestedExtension string) (*Parameters, error)
}

type RegexTokenizer struct {
	checks []check
	validExtensions []string
	cfg *Config
}

func (rt *RegexTokenizer) Tokenize(requestedTransformations, requestedExtension string) (*Parameters, error) {
	segments := strings.Split(strings.Trim(requestedTransformations, "/ "), "_")

	vErr := NewValidationError()
	if len(segments) == 0 {
		vErr.Add("segments", "no segments provided")
		return nil, vErr
	}

	if ! rt.isValidExtension(requestedExtension) {
		vErr.Add("extension", fmt.Sprintf("unsupported extension %s", requestedExtension))
		return nil, vErr
	}

	ds := &Parameters{segments: make(map[prefix]int)}
	for _, s := range segments {
		for _, check := range rt.checks {
			v, err := matchInteger(check.rx, s, check.min, check.max)
			if err != nil {
				vErr.Add(check.name, err.Error())
				continue
			} else if v != 0 && v != check.defaultValue {
				ds.segments[check.segment] = v
				continue
			}
		}
	}

	if ds.Empty() {
		vErr.Add("segments", "no valid segments provided")
		return nil, vErr
	}

	if !vErr.Empty() {
		return nil, vErr
	}

	if !ds.WantsTransformation() {
		ds.slug = "original"
	}

	if requestedExtension == "jpeg" {
		ds.extension = "jpg"
	} else {
		ds.extension = requestedExtension
	}

	return ds, nil
}

type check struct {
	name string
	rx *regexp.Regexp
	min int
	max int
	defaultValue int
	segment prefix
}

func NewRegexTokenizer(cfg *Config) *RegexTokenizer {
	checks := []check{
		{
			name: "height",
			rx: regexp.MustCompile(`^h(\d{1,4})$`),
			min: minHeight,
			max: maxHeight,
			segment: height,
		},
		{
			name: "width",
			rx: regexp.MustCompile(`^w(\d{1,4})$`),
			min: minWidth,
			max: maxWidth,
			segment: width,
		},
		{
			name:         "scale",
			rx:           regexp.MustCompile(`^s(\d{1,3})$`),
			min:          minScale,
			max:          maxScale,
			defaultValue: maxScale,
			segment:      scale,
		},
		{
			name: "quality",
			rx: regexp.MustCompile(`^q(\d{1,3})$`),
			min: minQuality,
			max: maxQuality,
			defaultValue: maxQuality,
			segment: quality,
		},
		{
			name: "opacity",
			rx: regexp.MustCompile(`^o(\d{1,3})$`),
			min: 1,
			max: 100,
			defaultValue: 100,
			segment: opacity,
		},
	}

	return &RegexTokenizer{
		checks: checks,
		validExtensions: []string{"jpg", "jpeg", "png"},
		cfg: cfg,
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

func (rt *RegexTokenizer) isValidExtension(ext string) bool {
	for _, vExt := range rt.validExtensions {
		if ext == vExt {
			return true
		}
	}

	return false
}


