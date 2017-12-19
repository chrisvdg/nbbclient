package nbd

import (
	"fmt"
	"log"
	"net"

	"github.com/chrisvdg/nbbclient/nbd/backend"
)

// NewServer returns a new server
func NewServer(backend backend.Backend) *Server {
	return &Server{
		Backend: backend,
	}
}

// Server represents an NBD server
type Server struct {
	Backend backend.Backend
}

// ListenAndServe starts listening for requests and serves them
func (s *Server) ListenAndServe(address string) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	// handle connections
	for {
		plainConn, err := l.Accept()
		if err != nil {
			log.Printf("Something went wrong serving the connection: %s\n", err)
		}
		fmt.Printf("Accepted connection from %s\n", plainConn.RemoteAddr().String())

		conn, err := NewConn(plainConn)
		defer conn.Close()

		err = conn.OldNegotiation(s.Backend.Size())
		if err != nil {
			fmt.Printf("Something went wrong negotiating: %s\n", err)
			return err
		}

		fmt.Println("Done negotiating")

		conn.HandleRequests()
	}
}
