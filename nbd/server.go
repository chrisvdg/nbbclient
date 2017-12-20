package nbd

import (
	"fmt"
	"log"
	"net"

	"github.com/chrisvdg/nbdserver/nbd/backend"
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
			log.Printf("Something went wrong accepting the connection: %s\n", err)
		}
		fmt.Printf("Accepted connection from %s\n", plainConn.RemoteAddr().String())

		conn, err := NewConn(plainConn, s.Backend)

		/* Old style negotiation
		err = conn.OldNegotiation(s.Backend.Size())
		if err != nil {
			fmt.Printf("Something went wrong negotiating: %s\n", err)
			return err
		}
		*/

		// Fixed newstyle negotiation
		name, err := conn.Negotiate(s.Backend.Size())
		if err != nil {
			fmt.Printf("Something went wrong negotiating: %s\n", err)
			return err
		}

		fmt.Println("Done negotiating")
		fmt.Printf("got vdiskID: %s\n", name)

		conn.HandleRequests()
		conn.Close()
	}
}
