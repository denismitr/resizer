package media

import "github.com/pkg/errors"

var ErrInvalidExtension = errors.New("invalid extension")

const (
	JPEG Extension = "jpg"
	PNG  Extension = "png"
	TIFF Extension = "tiff"
	WEBP Extension = "webp"
)

var extensions = map[string]Extension{
	"png":  Extension("png"),
	"jpg":  Extension("jpg"),
	"jpeg": Extension("jpg"),
	"webp": Extension("webp"),
}

var mimes = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	// fixme: webp
}

func GuessMimeFromExtension(ext string) (string, error) {
	if m, ok := mimes[ext]; ok {
		return m, nil
	}

	return "", errors.Wrapf(ErrInvalidExtension, "mime type unsupported for %s", ext)
}

func NormalizeExtension(ext string) (Extension, error) {
	if e, ok := extensions[ext]; ok {
		return e, nil
	}

	return "", errors.Wrapf(ErrInvalidExtension, "extension unsupported: %s", ext)
}
