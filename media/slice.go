package media

import "time"

type Slice struct {
	ID        ID
	ImageID   ID
	Width     int
	Height    int
	Size      int
	Name      string
	Bucket    string
	Format    string
	CreatedAt time.Time
	IsValid   bool
}
