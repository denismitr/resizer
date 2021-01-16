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

type Sort struct {
	By  string
	Asc bool
}

const DefaultPerPage = 25

type Pagination struct {
	Page    uint
	PerPage uint
}

func (p Pagination) Limit() uint {
	if p.PerPage == 0 {
		return DefaultPerPage
	}

	return p.PerPage
}

func (p Pagination) Offset() uint {
	if p.Page < 2 {
		return 0
	}

	return (p.Page - 1) * p.Limit()
}

type ImageFilter struct {
	Namespace     string
	OnlyPublished bool
	Sort          Sort
	Pagination
}

type Image struct {
	ID            ID         `json:"id"`
	Name          string     `json:"name"`
	OriginalName  string     `json:"originalName"`
	OriginalExt   string     `json:"originalExt"`
	OriginalSize  int        `json:"originalSize"`
	PublishAt     *time.Time `json:"publishAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Namespace     string     `json:"namespace"`
	OriginalSlice *Slice     `json:"originalSlice,omitempty"`
	Slices        Slices     `json:"slices"`
}

type Meta struct {
	Total   uint `json:"total"`
	Page    uint `json:"page"`
	PerPage uint `json:"perPage"`
}

type ImageCollection struct {
	Images []Image `json:"data"`
	Meta   Meta    `json:"meta"`
}
