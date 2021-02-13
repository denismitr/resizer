package proxy

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"github.com/denismitr/resizer/internal/manipulator"
	"github.com/denismitr/resizer/internal/media"
	"github.com/denismitr/resizer/internal/registry"
	"github.com/denismitr/resizer/internal/storage"
	"time"
)

var ErrResourceNotFound = errors.New("requested resource not found")
var ErrInternalError = errors.New("imageProxy error")
var ErrBadInput = errors.New("bad user input")

type ImageProxy interface {
	Proxy(
		ctx context.Context,
		dst io.Writer,
		transformation *manipulator.Transformation,
		img *media.Image,
	) (*media.Slice, error)

	Prepare(
		ctx context.Context,
		ID, requestedTransformations, ext string,
	) (*manipulator.Transformation, *media.Image, error)
}

type OnTheFlyPersistingImageProxy struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator *manipulator.Manipulator
	logger      *logrus.Logger
}

func (p *OnTheFlyPersistingImageProxy) Prepare(
	ctx context.Context,
	ID, requestedTransformations, ext string,
) (*manipulator.Transformation, *media.Image, error) {
	// Step !: tokenize request for transformation
	transformation, err := p.manipulator.Convert(requestedTransformations, ext)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: fetch image metadata and the original slice data from the Registry
	img, err := p.registry.GetImageByID(ctx, media.ID(ID), true)
	if err != nil {
		if errors.Is(err, registry.ErrEntityNotFound) {
			return nil, nil, errors.Wrapf(ErrResourceNotFound, "image with ID %v not found: %v", ID, err)
		}

		if errors.Is(err, registry.ErrInvalidID) {
			return nil, nil, errors.Wrap(ErrBadInput, err.Error())
		}

		return nil, nil, errors.Wrap(ErrInternalError, err.Error())
	}

	// Step 3: parse transformation parameters, applying the image specific constraints and settings
	if err := p.manipulator.Normalize(transformation, img); err != nil {
		return nil, nil, err
	}

	return transformation, img, nil
}

// fixme: return transformation
func (p *OnTheFlyPersistingImageProxy) Proxy(
	ctx context.Context,
	dst io.Writer,
	transformation *manipulator.Transformation,
	img *media.Image,
) (*media.Slice, error) {
	// Step 4: fetch an appropriate slice from the storage
	slice, exactMatch := p.fetchAppropriateSlice(ctx, img, img.ID.String()+"/"+transformation.Filename()) // fixme
	if slice == nil {
		panic("how can slice be nil at this point?")
	}

	// all the following operations are async generators
	errCh := make(chan error, 2)

	// Step 5: download slice file contents into a stream
	contents := p.getContentStream(ctx, slice, errCh)
	defer contents.Close()

	var doneCh <-chan *media.Slice

	// Step 6: if a matching file exists in the storage - stream it to the client
	// otherwise take the original slice, transform it, stream it to the client
	// and then asynchronously save it to the storage and registry for future use
	if exactMatch {
		doneCh = p.streamWithoutTransformation(dst, contents, slice, img, errCh)
	} else {
		doneCh = p.streamWithTransformation(dst, contents, img, transformation, errCh)
	}

	// wait for whatever happens first:
	// or we have a successful operation result and we can return response to client
	// or we have an error, or timeout happens
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		case transformedSlice := <-doneCh:
			if transformedSlice != nil {
				// TODO: prometheus monitoring
				//fmt.Printf("%#v", metadata)
				return transformedSlice, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (p *OnTheFlyPersistingImageProxy) streamWithoutTransformation(
	dst io.Writer,
	contents io.Reader,
	slice *media.Slice,
	img *media.Image,
	errCh chan<- error,
) <-chan *media.Slice {
	doneCh := make(chan *media.Slice)

	go func() {
		defer func() {
			close(doneCh)
		}()

		if _, err := io.Copy(dst, contents); err != nil {
			// fixme: normal error
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "error copying bytes").Error()}
		} else {
			mime, err := media.GuessMimeFromExtension(slice.Extension)
			if err != nil {
				// fixme: normal error
				errCh <- &httpError{statusCode: 400, message: errors.Wrap(err, "data invalid").Error()}
				return
			}

			doneCh <- &media.Slice{
				Filename:     slice.Filename,
				Mime:         mime,
				Width:        slice.Width,
				Height:       slice.Height,
				Namespace:    img.Namespace,
				Extension:    slice.Extension,
				Size:         slice.Size,
				ImageID:      slice.ImageID,
			}
		}
	}()

	return doneCh
}

func (p *OnTheFlyPersistingImageProxy) streamWithTransformation(
	dst io.Writer,
	contents io.Reader,
	img *media.Image,
	transformation *manipulator.Transformation,
	errCh chan<- error,
) <-chan *media.Slice {
	sliceCh, r := p.launchTransformation(contents, img, transformation, errCh)

	buf := &bytes.Buffer{}
	tr := io.TeeReader(r, buf)
	doneCh := make(chan *media.Slice)

	go func() {
		defer close(doneCh)

		if _, err := io.Copy(dst, tr); err != nil {
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "could not transform file").Error()}
			return
		}

		slice := <-sliceCh
		slice.ImageID = img.ID
		// fixme: more specific changes

		go p.saveTransformedSlice(slice, buf)

		doneCh <- slice
	}()

	return doneCh
}

func (p *OnTheFlyPersistingImageProxy) launchTransformation(
	contents io.Reader,
	img *media.Image,
	transformation *manipulator.Transformation,
	errCh chan<- error,
) (<-chan *media.Slice, io.Reader) {
	pr, pw := io.Pipe()
	sliceCh := make(chan *media.Slice)

	go func() {
		// conduct the transformations of the stream
		transformed, err := p.manipulator.Transform(contents, pw, transformation)
		errCh <- pw.Close()
		if err != nil {
			errCh <- errors.Wrapf(err, "could not transform image %s to %s", img.ID.String(), transformation.Filename())
			return
		}

		sliceCh <- transformed
	}()

	return sliceCh, pr
}

func (p *OnTheFlyPersistingImageProxy) saveTransformedSlice(slice *media.Slice, source io.Reader) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	slice.ID = p.registry.GenerateID()
	slice.IsValid = true
	slice.IsOriginal = false
	slice.Status = media.Active
	slice.CreatedAt = time.Now()

	item, err := p.storage.Put(ctx, slice.Namespace, slice.Filename, source)
	if err != nil {
		p.logger.Errorln(err)
		return
	}

	slice.Path = item.Path

	if _, err := p.registry.CreateSlice(ctx, slice); err != nil {
		// todo: delete from storage
		p.logger.Errorln(err)
	}
}

// fetchAppropriateSlice - fetches slice by filename and whether it exactly matches
// requested format and extension
func (p *OnTheFlyPersistingImageProxy) fetchAppropriateSlice(
	ctx context.Context,
	img *media.Image,
	filename string,
) (*media.Slice, bool) {
	slice, err := p.registry.GetSliceByImageIDAndFilename(ctx, img.ID, filename)
	if err != nil {
		p.logger.Errorln(err)
		p.logger.Errorln(img.OriginalSlice.Filename + " " + filename)
		return img.OriginalSlice, img.OriginalSlice.Filename == filename
	}

	return slice, true
}

func (p *OnTheFlyPersistingImageProxy) getContentStream(
	ctx context.Context,
	slice *media.Slice,
	errCh chan<- error,
) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer func() {
			if err := pw.Close(); err != nil {
				errCh <- err
			}
		}()

		if err := p.storage.Download(ctx, pw, slice.Namespace, slice.Filename); err != nil {
			errCh <- errors.Wrapf(
				err,
				"could not download file %s from namespace %s",
				slice.Filename, slice.Namespace)
		}
	}()

	return pr
}

func NewOnTheFlyPersistingImageProxy(
	l *logrus.Logger,
	r registry.Registry,
	s storage.Storage,
	m *manipulator.Manipulator,
) *OnTheFlyPersistingImageProxy {
	return &OnTheFlyPersistingImageProxy{
		registry:    r,
		storage:     s,
		manipulator: m,
		logger:      l,
	}
}
