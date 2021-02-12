package media

var mimes = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	// fixme: webp
}

func GuessMimeFromExtension(ext string) string {
	if m, ok := mimes[ext]; ok {
		return m
	}

	return "image/jpeg"
}
