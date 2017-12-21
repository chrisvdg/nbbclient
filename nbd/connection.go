package nbd

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/chrisvdg/nbdserver/nbd/backend"
	"github.com/pkg/errors"
)

// NewConn returns a new Connection
func NewConn(plainconn net.Conn, backend backend.Backend) (*Connection, error) {
	conn := &Connection{
		plainconn: plainconn,
		backend:   backend,
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
		rh := nbdReply{
			NbdReplyMagic: NBD_REPLY_MAGIC,
			NbdHandle:     req.NbdHandle,
			NbdError:      0,
		}

		switch req.NbdCommandType {
		case NBD_CMD_READ:
			fmt.Println("received read command")
			// read from backend
			data, err := c.backend.ReadAt(nil, int64(req.NbdLength), int64(req.NbdLength))
			if err != nil {
				fmt.Printf("Something went wrong reading from backend: %v", err)
				rh.NbdError = NBD_EIO
			}

			// send reply header
			binary.Write(c.plainconn, binary.BigEndian, &rh)

			// send data if no error occurred
			if rh.NbdError == 0 {
				binary.Write(c.plainconn, binary.BigEndian, data)
			}
		case NBD_CMD_WRITE:
			fmt.Println("received write command")
			// read data from request and write to backend
			buf := make([]byte, req.NbdLength)
			binary.Read(c.plainconn, binary.BigEndian, &buf)

			_, err := c.backend.WriteAt(nil, buf, int64(req.NbdOffset))
			if err != nil {
				fmt.Printf("Something went wrong writing to the backend: %v", err)
				rh.NbdError = NBD_EIO
			}

			if rh.NbdError == 0 {
				err = c.backend.Flush(nil)
				if err != nil {
					fmt.Printf("Something went wrong flushing the backend: %v", err)
					rh.NbdError = NBD_EIO
				}
			}

			binary.Write(c.plainconn, binary.BigEndian, &rh)
		case NBD_CMD_FLUSH:
			fmt.Println("received flush command")
			err = c.backend.Flush(nil)
			if err != nil {
				fmt.Printf("flushing the backend failed: %s\n", err)
				rh.NbdError = NBD_EIO
			}

			binary.Write(c.plainconn, binary.BigEndian, &rh)
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

// Negotiate executes a fixed-newstyle negotiation
func (c *Connection) Negotiate(exportSize uint64) (string, error) {
	// Send fixed-newstyle header
	nsh := nbdNewStyleHeader{
		NbdMagic:       NBD_MAGIC,
		NbdOptsMagic:   NBD_OPTS_MAGIC,
		NbdGlobalFlags: NBD_FLAG_FIXED_NEWSTYLE,
	}

	fmt.Println("sending header")
	err := binary.Write(c.plainconn, binary.BigEndian, nsh)
	if err != nil {
		return "", err
	}

	// Read client flags
	fmt.Println("Reading client flags")
	var clf nbdClientFlags
	err = binary.Read(c.plainconn, binary.BigEndian, &clf)
	if err != nil {
		return "", err
	}

	// Haggle client options
	fmt.Println("Starting to haggle")
	done := false
	name := ""
	for !done {
		var opt nbdClientOpt
		err = binary.Read(c.plainconn, binary.BigEndian, &opt)
		if err != nil {
			return "", err
		}

		switch opt.NbdOptID {
		// this option also terminates a negotiation
		case NBD_OPT_EXPORT_NAME:
			// read name
			nameBS := make([]byte, opt.NbdOptLen)
			n, err := io.ReadFull(c.plainconn, nameBS)
			if err != nil {
				return "", err
			}
			if uint32(n) != opt.NbdOptLen {
				return "", errors.New("received incomplete name")
			}

			// validate export name
			name = string(nameBS)

			// export details
			ed := nbdExportDetails{
				NbdExportSize:  exportSize,
				NbdExportFlags: 333,
			}
			err = binary.Write(c.plainconn, binary.BigEndian, ed)
			if err != nil {
				return "", fmt.Errorf("something went wrong sending export details: %v", err)
			}

			// empty zeroes
			if clf.NbdClientFlags&NBD_FLAG_C_NO_ZEROES == 0 {
				// send 124 bytes of zeroes.
				zeroes := make([]byte, 124, 124)
				if err := binary.Write(c.plainconn, binary.BigEndian, zeroes); err != nil {
					return "", fmt.Errorf("Could not write zeroes: %v", err)
				}
			}

			done = true
		default:
			err := skip(c.plainconn, opt.NbdOptLen)
			if err != nil {
				return "", err
			}

			// unsupported optID
			or := nbdOptReply{
				NbdOptReplyMagic:  NBD_REP_MAGIC,
				NbdOptID:          opt.NbdOptID,
				NbdOptReplyType:   NBD_REP_ERR_UNSUP,
				NbdOptReplyLength: 0,
			}
			if err := binary.Write(c.plainconn, binary.BigEndian, or); err != nil {
				return "", fmt.Errorf("Cannot reply to unsupported option %s", err)
			}
		}

	}

	return name, nil
}

// skip bytes
func skip(r io.Reader, n uint32) error {
	for n > 0 {
		l := n
		if l > 1024 {
			l = 1024
		}
		b := make([]byte, l)
		if nr, err := io.ReadFull(r, b); err != nil {
			return err
		} else if nr != int(l) {
			return errors.New("skip returned short read")
		}
		n -= l
	}
	return nil
}

// Close closes the connection
func (c *Connection) Close() {
	c.plainconn.Close()
}
