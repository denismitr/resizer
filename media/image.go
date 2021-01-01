package media

type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) None() bool {
	return id == ""
}

type Image struct {
	ID ID
	OriginalName string
	Bucket string
	Path string
}
