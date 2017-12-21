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
	blockSize     = 1024
)

func main() {
	// create temp file
	file, err := ioutil.TempFile(os.TempDir(), "nbdfile")
	if err != nil {
		log.Fatal(err)
	}

	// give file a size
	_, err = file.Write(make([]byte, 1024*1024*1024))
	if err != nil {
		log.Fatal(err)
	}

	// capture interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		// cleanup file
		file.Close()
		os.Remove(file.Name())
		os.Exit(1)
	}()

	// create backend
	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	backend := backend.NewFile(file, uint64(stat.Size()))

	// start server
	server := nbd.NewServer(backend)
	fmt.Printf("NBD server listening on: `%s`\n", listenAddress)
	fmt.Printf("With backend file: `%s` of size %d bytes\n", file.Name(), stat.Size())
	err = server.ListenAndServe(listenAddress)
	if err != nil {
		log.Fatal(err)
	}
}
