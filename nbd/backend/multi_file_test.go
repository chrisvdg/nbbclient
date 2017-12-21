package backend

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// test data
var (
	helloWorld     = []byte("Hello world!")
	helloWorldLen  = len(helloWorld)
	lorumImpsum    = []byte("Lorum Ipsum")
	lorumImpsumLen = len(lorumImpsum)
)

func TestMultiFile(t *testing.T) {
	require := require.New(t)

	files, err := generateFiles(2)
	require.NoError(err, "Failed to generate test files")
	defer cleanupFiles(files)
	b := NewMultiFile(files, 4096)

	// write to files through the backend and read from it again
	for i, f := range files {
		address := int64(i << 24)

		// write to file
		l, err := f.WriteAt(helloWorld, 0)
		require.NoError(err)
		require.Equal(helloWorldLen, int(l))

		// read from backend
		d, err := b.ReadAt(nil, address, int64(helloWorldLen))
		require.NoError(err)
		require.Equal(helloWorld, d)

		// write to backend
		wrLen, err := b.WriteAt(nil, lorumImpsum, address)
		require.NoError(err)
		require.Equal(lorumImpsumLen, int(wrLen))

		// read from file
		d = make([]byte, lorumImpsumLen)
		_, err = f.ReadAt(d, 0)
		require.NoError(err)
		require.Equal(lorumImpsum, d)
	}
}

func TestMultiFile_GetFile(t *testing.T) {
	require := require.New(t)

	// create new Multifile backend with a few files
	files, err := generateFiles(3)
	require.NoError(err, "Failed to generate test files")
	defer cleanupFiles(files)

	b := NewMultiFile(files, 0)

	// test valid file addresses
	addr := int64(0)
	f, err := b.getFile(addr)
	require.NoError(err, "failed to get file with valid file address")
	require.Equal(files[0].Name(), f.Name(), "file with file address 0 should return the file in the backend with index 0")

	addr = int64(1) << 24 // set 4th byte to 1
	f, err = b.getFile(addr)
	require.NoError(err, "failed to get file with valid file address")
	require.Equal(files[1].Name(), f.Name(), "file with file address 1 should return the file in the backend with index 1")

	addr = int64(2) << 24
	f, err = b.getFile(addr)
	require.NoError(err, "failed to get file with valid file address")
	require.Equal(files[2].Name(), f.Name(), "file with file address 2 should return the file in the backend with index 2")

	// fetch a file that's out of range
	addr = int64(3) << 24
	f, err = b.getFile(addr)
	require.Error(err, "this backend file should not be available")
}

func generateFiles(n int) ([]*os.File, error) {
	var files []*os.File
	for i := 0; i < n; i++ {
		file, err := ioutil.TempFile(os.TempDir(), "nbd_test_file")
		if err != nil {
			cleanupFiles(files)
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

func cleanupFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
		os.Remove(f.Name())
	}
}
