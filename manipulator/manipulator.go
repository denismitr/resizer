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
	transformationsPool sync.Pool
	imageTransformer    *ImageTransformer
	paramConverter      *ParamConverter
	normalizer          *Normalizer
}

func New(cfg *Config) *Manipulator {
	return &Manipulator{
		transformationsPool: sync.Pool{
			New: func() interface{} { return new(Transformation) },
		},
		imageTransformer: &ImageTransformer{cfg: cfg},
		normalizer:       &Normalizer{cfg: cfg},
		paramConverter:   NewParamConverter(cfg),
	}
}

func (m *Manipulator) Normalize(t *Transformation, img *media.Image) error {
	if err := m.normalizer.Normalize(t, img); err != nil {
		return err
	}

	return nil
}

func (m *Manipulator) Convert(requestedTransformation, requestedExtension string) (*Transformation, error) {
	var t *Transformation
	t = m.transformationsPool.Get().(*Transformation)

	if err := m.paramConverter.ConvertTo(t, requestedTransformation, requestedExtension); err != nil {
		return t, err
	}

	return t, nil
}

func (m *Manipulator) Transform(source io.Reader, dst io.Writer, t *Transformation) (*Result, error) {
	// TODO: return transformation to memory pool
	return m.imageTransformer.Transform(source, dst, t)
}

func (m *Manipulator) Reset(t *Transformation) {
	t.Reset()
	m.transformationsPool.Put(t)
}

type PoolManipulator struct {
	transformationsPool sync.Pool
	imageTransformer    *ImageTransformer
	paramConverter      *ParamConverter
}

type Result struct {
	Width     int
	Height    int
	Extension string
	Size      int
	Quality   Percent
}

func (r *Result) OriginalFilename() string {
	if r.Width == 0 || r.Height == 0 || r.Extension == "" {
		panic(fmt.Sprintf("how can result %v be missing required parts", r))
	}

	return fmt.Sprintf("%s%d_%s%d.%s", HeightPrefix, r.Height, WidthPrefix, r.Width, r.Extension)
}