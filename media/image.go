package media

import "time"

type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) None() bool {
	return id == ""
}

type Image struct {
	ID           ID
	OriginalName string
	OriginalExt  string
	OriginalSize int
	PublishAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Bucket       string
	Path         string
	Url          string
}
