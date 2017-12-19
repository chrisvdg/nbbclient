package backend

import "context"

// Backend represents an NBD backend
type Backend interface {
	Size() uint64
	WriteAt(ctx context.Context, b []byte, offset int64) (int64, error)
	ReadAt(ctx context.Context, offset, length int64) ([]byte, error)
	Flush(ctx context.Context) error
	Close(ctx context.Context) error
}
