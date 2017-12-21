package backend

import (
	"context"
	"errors"
	"os"
)

const (
	addressInFileMask = 0x00ffffff
)

// NewMultiFile returns a new backend that has multiple files
func NewMultiFile(files []*os.File, totalSize uint64) *MultiFile {
	return &MultiFile{
		files: files,
		size:  totalSize,
	}
}

// MultiFile represents a multiple file backend
//
// A multifile backend uses the first byte of the block address to identify
// the file to write to while the last 3 bytes indicate
// the position within that file.
type MultiFile struct {
	files []*os.File
	size  uint64
}

// Size implements Backend.Size
func (f *MultiFile) Size() uint64 {
	return f.size
}

// WriteAt implements Backend.WriteAt
func (f *MultiFile) WriteAt(ctx context.Context, b []byte, offset int64) (int64, error) {
	file, err := f.getFile(offset)
	if err != nil {
		return 0, err
	}

	n, err := file.WriteAt(b, offset&addressInFileMask)

	return int64(n), err
}

// ReadAt implements Backend.ReadAt
func (f *MultiFile) ReadAt(ctx context.Context, offset, length int64) ([]byte, error) {
	file, err := f.getFile(offset)
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, length)
	_, err = file.ReadAt(bytes, offset&addressInFileMask)

	return bytes, err
}

// Flush implements Backend.Flush
func (f *MultiFile) Flush(ctx context.Context) error {
	for _, f := range f.files {
		err := f.Sync()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFile returns the file corresponding to the first byte of the address
func (f *MultiFile) getFile(reqAddress int64) (*os.File, error) {
	fileAddr := reqAddress >> 24

	if int(fileAddr) >= len(f.files) {
		return nil, errors.New("Invalid file address")
	}

	return f.files[fileAddr], nil
}
