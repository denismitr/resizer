package media

import (
	"strings"
	"time"
)

type ID string

func (id ID) String() string {
	return string(id)
}

func (id ID) None() bool {
	return id == ""
}

type Actions string
type Extension string

type Image struct {
	ID           ID
	OriginalName string
	OriginalExt  string
	OriginalSize int
	PublishAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Bucket       string
	Path         string
	Url          string
	Name         string
	Slices       []Slice
}

func (img Image) GetFileNameFromPath() string {
	return strings.TrimSuffix(img.Bucket+"/", img.Path) // fixme
}
