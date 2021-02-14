package media

import (
	"time"
)

type Status string

const (
	Unsaved    Status = "unsaved"
	Pending    Status = "pending"
	Processing Status = "processing"
	Retrying   Status = "retrying"
	Active     Status = "active"
)

type Slices []Slice

type Slice struct {
	ID      ID  `json:"id"`
	ImageID ID  `json:"imageId"`
	Width   int `json:"width"`
	Height  int `json:"height"`
	Size    int `json:"size"`
	Quality int `json:"quality"`

	// imageID/filename
	Filename  string `json:"filename"`
	Namespace string `json:"namespace"`

	// Path in storage (in S3 bucket/filename)
	Path string `json:"path"`

	// Flag that shows that p
	Cropped bool `json:"cropped"`

	// Extension is denormalized for querying
	Extension Extension    `json:"extension"`
	Mime      string    `json:"mime"`
	CreatedAt time.Time `json:"createdAt"`
	IsValid   bool      `json:"-"`
	Status    Status    `json:"status"`

	// IsOriginal - originally uploaded image
	IsOriginal bool `json:"isOriginal"`
}

func ComputeSliceFilename(imageID ID, filename string) string {
	return imageID.String() + "/" + filename
}

func ComputeSlicePath(imageNamespace string, imageID ID, filename string) string {
	return imageNamespace + "/" + imageID.String() + "/" + filename
}
