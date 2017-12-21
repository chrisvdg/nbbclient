package backend

import (
	"context"
	"os"
)

// NewFile returns a single file backend
func NewFile(file *os.File, size uint64) *File {
	return &File{
		file: file,
		size: size,
	}
}

// File represents a single file backend
type File struct {
	file *os.File
	size uint64
}

// Size implements Backend.Size
func (f *File) Size() uint64 {
	return f.size
}

// WriteAt implements Backend.WriteAt
func (f *File) WriteAt(ctx context.Context, b []byte, offset int64) (int64, error) {
	n, err := f.file.WriteAt(b, offset)

	return int64(n), err
}

// ReadAt implements Backend.ReadAt
func (f *File) ReadAt(ctx context.Context, offset, length int64) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := f.file.ReadAt(bytes, offset)

	return bytes, err
}

// Flush implements Backend.Flush
func (f *File) Flush(ctx context.Context) error {
	return f.file.Sync()
}

// Close implements Backend.Close
func (f *File) Close(ctx context.Context) error {
	return f.file.Close()
}
