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
	Put(ctx context.Context, bucket, filename string, source io.ReadSeeker) (*Item, error)
	Download(ctx context.Context, dst io.WriterAt, bucket, file string) error
}
