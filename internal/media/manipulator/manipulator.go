package manipulator

import (
	"github.com/denismitr/resizer/internal/media"
	"github.com/pkg/errors"
	"io"
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

func (m *Manipulator) NormalizeTransformation(t *Transformation, img *media.Image) error {
	if err := m.normalizer.normalize(t, img); err != nil {
		return err
	}

	return nil
}

// CreateTransformation - converts transformation request into transformation object
func (m *Manipulator) CreateTransformation(transformationRequest, requestedExtension string) (*Transformation, error) {
	t := new(Transformation)

	if err := m.paramConverter.convertTo(t, transformationRequest, requestedExtension); err != nil {
		return t, err
	}

	return t, nil
}

func (m *Manipulator) CreateSlice(
	source io.Reader,
	dst io.Writer,
	rootImage *media.Image,
	t *Transformation,
) (*media.Slice, error) {
	// TODO: return transformation to memory pool
	return m.imageTransformer.createSlice(source, dst, rootImage, t)
}

func (m *Manipulator) CreateOriginalSlice(
	source io.Reader,
	dst io.Writer,
	rootImage *media.Image,
) (*media.Slice, error) {
	// TODO: return transformation to memory pool
	return m.imageTransformer.createOriginalSlice(source, dst, rootImage)
}

type PoolManipulator struct {
	transformationsPool sync.Pool
	imageTransformer    *imageTransformer
	paramConverter      *paramConverter
}
