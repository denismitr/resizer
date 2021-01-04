package proxy

import (
	"fmt"
	"regexp"
	"resizer/manipulator"
	"resizer/media"
	"strconv"
	"strings"
)

type parser struct {
	rxHeight     *regexp.Regexp
	rxWidth      *regexp.Regexp
	rxQuality    *regexp.Regexp
	rxProportion *regexp.Regexp
}

func newParser() *parser {
	return &parser{
		rxHeight:     regexp.MustCompile(`^h(\d{1,4})$`),
		rxWidth:      regexp.MustCompile(`^w(\d{1,4})$`),
		rxQuality:    regexp.MustCompile(`^q(\d{1,3})$`),
		rxProportion: regexp.MustCompile(`^parser(\d{1,3})$`),
	}
}

func (p *parser) createTransformation(img *media.Image, requestedTransformations, extension string) (*manipulator.Transformation, error) {
	segments := strings.Split(strings.Trim(requestedTransformations, "/ "), "_")

	t := manipulator.Transformation{}

	for _, s := range segments {
		var match []string
		match = p.rxHeight.FindStringSubmatch(s)
		if match != nil && len(match) > 1 {
			// todo: check range
			height, err := strconv.Atoi(match[1])
			if err == nil && height != 0 {
				t.Resize.Height = manipulator.Pixels(height)
			}

			continue
		}

		match = p.rxWidth.FindStringSubmatch(s)
		if match != nil && len(match) > 1 {
			width, err := strconv.Atoi(match[1])
			// todo: check range
			if err == nil && width != 0 {
				t.Resize.Width = manipulator.Pixels(width)
			}

			continue
		}

		match = p.rxQuality.FindStringSubmatch(s)
		if match != nil && len(match) > 1 {
			quality, err := strconv.Atoi(match[1])
			// todo: check range
			if err == nil {
				t.Quality = manipulator.Percent(quality)
			}
			continue
		}

		match = p.rxProportion.FindStringSubmatch(s)
		if match != nil && len(match) > 1 {
			proportion, err := strconv.Atoi(match[1])
			// todo: check range
			if err == nil {
				t.Resize.Proportion = manipulator.Percent(proportion)
			}
			continue
		}
	}

	switch extension {
	case "png":
		t.Format = manipulator.PNG
	case "jpg", "jpeg":
		t.Format = manipulator.JPEG
	default:
		return nil, httpError{
			statusCode: 422,
			message:    "The given data was invalid",
			details:    map[string]string{"format": fmt.Sprintf("Format %storage is unsupported", extension)},
		}
	}

	return &t, nil
}
