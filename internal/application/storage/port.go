package storage

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, key string, content io.Reader) (Location, error)
	Get(ctx context.Context, key string, loc Location) (io.ReadCloser, error)
	Delete(ctx context.Context, key string, loc Location) error
	Exists(ctx context.Context, key string, loc Location) (bool, error)
	List(ctx context.Context, prefix string, loc Location) ([]string, error)
}
