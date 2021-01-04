package storage

import (
	"context"
	"errors"
	"io"
)

var ErrStorageFailed = errors.New("storage error")

type Item struct {
	Result string
	Path string
	URL  string
}

type Storage interface {
	Put(ctx context.Context, bucket, filename string, source io.Reader) (*Item, error)
	Download(ctx context.Context, writer io.Writer, bucket, file string) error
}
