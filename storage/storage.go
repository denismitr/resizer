package storage

import (
	"context"
	"io"
)

type Item struct {
	Path string
	URL  string
}

type Storage interface {
	Put(ctx context.Context, bucket string, source io.Reader) (Item, error)
}
