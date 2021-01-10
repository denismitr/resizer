package media

import (
	"strings"
	"time"
)

type Status string

const (
	Pending    Status = "pending"
	Processing        = "processing"
	Retrying          = "retrying"
	Ready             = "ready"  // fixme: rename to active
)

type Slices []Slice

type Slice struct {
	ID       ID     `json:"id"`
	ImageID  ID     `json:"imageId"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Size     int    `json:"size"`
	// imageID/filename
	Filename string `json:"filename"`
	Bucket string `json:"bucket"`

	// Path in storage (in S3 bucket/filename)
	Path string `json:"path"`

	// Extension is denormalized for querying
	Extension string    `json:"extension"`
	CreatedAt time.Time `json:"createdAt"`
	IsValid   bool      `json:"-"`
	Status    Status    `json:"status"`

	// IsOriginal - originally uploaded image
	IsOriginal bool `json:"isOriginal"`
}

func (s Slice) GetFileNameFromPath() string {
	return strings.TrimSuffix(s.Bucket+"/"+s.ImageID.String(), s.Path)
}

func ComputeSliceBucket(imageBucket string, imageID ID) string {
	if imageID.String() == "" {
		panic("no id")
	}
	return imageBucket + "/" + imageID.String()
}

func ComputeSliceFilename(imageID ID, filename string) string {
	return imageID.String() + "/" + filename
}

func ComputeSlicePath(imageBucket string, imageID ID, filename string) string {
	return imageBucket + "/" + imageID.String() + "/" + filename
}
