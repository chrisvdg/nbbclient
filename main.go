package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/chrisvdg/nbdserver/nbd"
	"github.com/chrisvdg/nbdserver/nbd/backend"
)

const (
	listenAddress = ":7777"
	exportName    = "default"
	totalSize     = 1024 * 1024 * 1024
)

func main() {
	// create backend
	files, err := generateFiles(totalSize)
	if err != nil {
		log.Fatal(err)
	}

	// capture interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// cleanup files
		cleanupFiles(files)
		os.Exit(1)
	}()

	backend := backend.NewMultiFile(files, uint64(totalSize))

	// start server
	server := nbd.NewServer(backend)
	fmt.Printf("NBD server listening on: `%s`\n", listenAddress)
	err = server.ListenAndServe(listenAddress)
	if err != nil {
		log.Fatal(err)
	}
}

// generateFiles generates temporary backend files
// returns a list of those files, the total memory and error
func generateFiles(size int) ([]*os.File, error) {
	var files []*os.File

	done := false
	sizeLeft := size
	for !done {
		fSize := sizeLeft

		// detect last iteration
		if sizeLeft < backend.MaxSingleFileSize {
			done = true
		} else {
			fSize = backend.MaxSingleFileSize
		}

		file, err := ioutil.TempFile(os.TempDir(), "nbd-file")
		if err != nil {
			cleanupFiles(files)
			return nil, err
		}
		// give files their size
		_, err = file.Write(make([]byte, fSize))
		if err != nil {
			cleanupFiles(files)
			return nil, err
		}

		files = append(files, file)

		sizeLeft = sizeLeft - fSize
	}

	return files, nil
}

func cleanupFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
		os.Remove(f.Name())
	}
}
