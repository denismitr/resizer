package media

import (
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
	ID            ID         `json:"id"`
	Name          string     `json:"name"`
	OriginalName  string     `json:"originalName"`
	OriginalExt   string     `json:"originalExt"`
	OriginalSize  int        `json:"originalSize"`
	PublishAt     *time.Time `json:"publishAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Bucket        string     `json:"bucket"`
	OriginalSlice *Slice     `json:"originalSlice,omitempty"`
}
