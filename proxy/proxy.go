package proxy

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"resizer/manipulator"
	"resizer/media"
	"resizer/registry"
	"resizer/storage"
	"time"
)

var ErrResourceNotFound = errors.New("requested resource not found")
var ErrInternalError = errors.New("proxy error")

type metadata struct {
	filename     string
	mime         string
	extension    string
	width        int
	height       int
	originalName string
	bucket       string
	size         int
	imageID      string
}

type ImageProxy interface {
	Proxy(ctx context.Context, dst io.Writer, ID, format, ext string) (*metadata, error)
}

type OnTheFlyPersistingImageProxy struct {
	registry    registry.Registry
	storage     storage.Storage
	manipulator manipulator.Manipulator
	parser      *media.Parser
}

// fixme: return transformation
func (p *OnTheFlyPersistingImageProxy) Proxy(
	ctx context.Context,
	dst io.Writer,
	ID, requestedTransformations, ext string,
) (*metadata, error) {
	// we fetch image and original slice metadata from the Registry
	img, err := p.registry.GetImageByID(ctx, media.ID(ID))
	if err != nil {
		if errors.Is(err, registry.ErrImageNotFound) || errors.Is(err, registry.ErrSliceNotFound) {
			return nil, errors.Wrapf(ErrResourceNotFound, "image with ID %v not found %v", ID, err)
		}

		return nil, errors.Wrap(ErrInternalError, err.Error())
	}

	// we parse the requested transformations string and extension
	transformation, err := p.parser.Parse(img, requestedTransformations, ext)
	if err != nil {
		if vErr, ok := err.(*media.ValidationError); ok {
			return nil, &httpError{
				statusCode: 422,
				message:    "The given data was invalid",
				details:    vErr.Errors(),
			}
		}

		return nil, err
	}

	errCh := make(chan error, 2)

	// fetch an appropriate slice from the storage
	slice, exactMatch := p.fetchAppropriateSlice(ctx, img, transformation.ComputeFilename())
	if slice == nil {
		panic("how can slice be nil at this point?")
	}

	// download slice contents into a stream
	contents := p.getContentStream(ctx, slice, errCh)
	defer contents.Close()

	var doneCh <-chan *metadata
	fmt.Fprintf(os.Stderr, "Exact match %v", exactMatch)
	if exactMatch {
		doneCh = p.streamWithoutTransformation(dst, contents, slice, img, errCh)
	} else {
		doneCh = p.streamWithTransformation(dst, contents, img, transformation, errCh)
	}

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		case metadata := <-doneCh:
			if metadata != nil {
				return metadata, nil
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
) <-chan *metadata {
	doneCh := make(chan *metadata)

	go func() {
		defer func() {
			close(doneCh)
		}()

		if _, err := io.Copy(dst, contents); err != nil {
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "error copying bytes").Error()}
		} else {
			doneCh <- &metadata{
				filename:     slice.Filename,
				mime:         createMimeFormExtension(slice.Extension),
				originalName: img.OriginalName,
				width: slice.Width,
				height: slice.Height,
				bucket: img.Bucket,
				extension: slice.Extension,
				size: slice.Size,
				imageID: slice.ImageID.String(),
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
) <-chan *metadata {
	metadataCh, r := p.launchTransformation(contents, img, transformation, errCh)

	buf := &bytes.Buffer{}
	tr := io.TeeReader(r, buf)
	doneCh := make(chan *metadata)

	go func() {
		defer close(doneCh)

		if _, err := io.Copy(dst, tr); err != nil {
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "could not transform file").Error()}
			return
		}

		metadata := <-metadataCh

		go p.saveTransformedSlice(metadata, buf)

		doneCh <- metadata
	}()

	return doneCh
}

func (p *OnTheFlyPersistingImageProxy) launchTransformation(
	contents io.Reader,
	img *media.Image,
	transformation *manipulator.Transformation,
	errCh chan <-error,
) (<-chan *metadata, io.Reader) {
	pr, pw := io.Pipe()
	metadataCh := make(chan *metadata)

	go func() {
		// conduct the transformations of the stream
		transformed, err := p.manipulator.Transform(contents, pw, transformation)
		errCh <- pw.Close()
		if err != nil {
			errCh <- &httpError{statusCode: 500, message: errors.Wrap(err, "could not transform file").Error()}
			return
		}

		metadataCh <- &metadata{
			filename:     transformation.ComputeFilename(),
			mime:         createMimeFormExtension(transformed.Extension),
			originalName: img.OriginalName,
			width: transformed.Width,
			height: transformed.Height,
			bucket: img.Bucket,
			extension: transformed.Extension,
			size: transformed.Size,
			imageID: img.ID.String(),
		}
	}()

	return metadataCh, pr
}

func (p *OnTheFlyPersistingImageProxy) saveTransformedSlice(metadata *metadata, source io.Reader) {
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	var slice media.Slice
	slice.ImageID = media.ID(metadata.imageID)
	slice.Filename = metadata.filename
	slice.Width = metadata.width
	slice.Height = metadata.height
	slice.Extension = metadata.extension
	slice.Size = metadata.size
	slice.Bucket = metadata.bucket
	slice.IsValid = true
	slice.IsOriginal = true
	slice.CreatedAt = time.Now()

	item, err := p.storage.Put(ctx, slice.Bucket, slice.Filename, source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err) // fixme: logrus
		return
	}

	slice.Path = item.Path

	if _, err := p.registry.CreateSlice(ctx, &slice); err != nil {
		// todo: delete from storage
		fmt.Fprintln(os.Stderr, err) // fixme: logrus
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
		fmt.Fprintln(os.Stderr, err) // fixme: logrus
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

		if err := p.storage.Download(ctx, pw, slice.Bucket, slice.Filename); err != nil {
			errCh <- errors.Wrapf(
				err,
				"could not download file %s from bucket %s",
				slice.Filename, slice.Bucket)
		}
	}()

	return pr
}

func NewOnTheFlyPersistingImageProxy(
	r registry.Registry,
	s storage.Storage,
	m manipulator.Manipulator,
	p *media.Parser,
) *OnTheFlyPersistingImageProxy {
	return &OnTheFlyPersistingImageProxy{
		registry:    r,
		storage:     s,
		manipulator: m,
		parser:      p,
	}
}
