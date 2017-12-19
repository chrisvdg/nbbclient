package nbd

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/chrisvdg/nbbclient/nbd/backend"
	"github.com/pkg/errors"
)

// NewConn returns a new Connection
func NewConn(plainconn net.Conn) (*Connection, error) {
	conn := &Connection{
		plainconn: plainconn,
	}

	return conn, nil
}

// Connection represents an NBD connection
type Connection struct {
	plainconn net.Conn
	backend   backend.Backend
}

// HandleRequests handles an nbd requests for a single connection
func (c *Connection) HandleRequests() {
	for {
		var req nbdRequest
		err := binary.Read(c.plainconn, binary.BigEndian, &req)
		if err != nil {
			if errors.Cause(err) == io.EOF {
				fmt.Println("Client closed connection abruptly")
			} else {
				fmt.Printf("Something went wrong reading a request: %v\n", err)
				fmt.Println(req)
			}

			return
		}

		if req.NbdRequestMagic != NBD_REQUEST_MAGIC {
			fmt.Printf("Client had bad magic number in request\n")
			return
		}

		switch req.NbdCommandType {
		case NBD_CMD_READ:
			fmt.Println("received read command")
			// read from backend and return reply
			//data, err := c.backend.ReadAt(nil, req.NbdLength, req.NbdLength)
		case NBD_CMD_WRITE:
			fmt.Println("received write command")
			// read data from request and write to backend

		default:
			fmt.Printf("unsupported command %d\n", req.NbdCommandType)
		}
	}
}

// OldNegotiation executes an oldstyle negotiation
func (c *Connection) OldNegotiation(exportSize uint64) error {
	osh := nbdOldStyleHeader{
		NbdMagic:        NBD_MAGIC,
		NbdCliservMagic: NBD_CLISERV_MAGIC,
		ExportSize:      exportSize,
		Flags:           0x3,
	}

	err := binary.Write(c.plainconn, binary.BigEndian, osh)
	if err != nil {
		return err
	}

	// send empty reserved bytes
	reserved := make([]byte, 124)
	return binary.Write(c.plainconn, binary.BigEndian, reserved)
}

// Negotiate executes a fixed newstyle negotiation
func (c *Connection) Negotiate() (string, error) {
	// Send fixed-newstyle header
	nsh := nbdNewStyleHeader{
		NbdMagic:       NBD_MAGIC,
		NbdOptsMagic:   NBD_OPTS_MAGIC,
		NbdGlobalFlags: NBD_FLAG_FIXED_NEWSTYLE,
	}

	err := binary.Write(c.plainconn, binary.BigEndian, nsh)
	if err != nil {
		return "", err
	}

	// Read client flags
	var clf nbdClientFlags
	err = binary.Read(c.plainconn, binary.BigEndian, &clf)
	if err != nil {
		return "", err
	}

	// Haggle client options

	// Send export details

	return "", nil
}

// Close closes the connection
func (c *Connection) Close() {
	c.plainconn.Close()
}
