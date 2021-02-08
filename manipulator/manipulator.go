package manipulator

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"resizer/media"
	"sync"
)

var ErrTransformationFailed = errors.New("manipulator bad transformation request")
var ErrBadImage = errors.New("manipulator bad image provided")

// maximum distance into image to look for EXIF tags
const maxExifSize = 1 << 20
const DefaultQuality = 100

type Manipulator struct {
	imageTransformer *imageTransformer
	paramConverter   *paramConverter
	normalizer       *normalizer
}

func New(cfg *Config) *Manipulator {
	return &Manipulator{
		imageTransformer: newImageTransformer(cfg),
		normalizer:       newNormalizer(cfg),
		paramConverter:   newParamConverter(cfg),
	}
}

func (m *Manipulator) Normalize(t *Transformation, img *media.Image) error {
	if err := m.normalizer.normalize(t, img); err != nil {
		return err
	}

	return nil
}

// Convert - converts transformation request into transformation object
func (m *Manipulator) Convert(transformationRequest, requestedExtension string) (*Transformation, error) {
	t := new(Transformation)

	if err := m.paramConverter.convertTo(t, transformationRequest, requestedExtension); err != nil {
		return t, err
	}

	return t, nil
}

func (m *Manipulator) Transform(source io.Reader, dst io.Writer, t *Transformation) (*Result, error) {
	// TODO: return transformation to memory pool
	return m.imageTransformer.transform(source, dst, t)
}

type PoolManipulator struct {
	transformationsPool sync.Pool
	imageTransformer    *imageTransformer
	paramConverter      *paramConverter
}

type Result struct {
	Width     int
	Height    int
	Extension string
	Size      int
	Cropped   bool
	Quality   Percent
}

func (r *Result) OriginalFilename() string {
	if r.Width == 0 || r.Height == 0 || r.Extension == "" {
		panic(fmt.Sprintf("how can result %v be missing required parts", r))
	}

	return fmt.Sprintf("%s%d_%s%d.%s", height, r.Height, width, r.Width, r.Extension)
}
