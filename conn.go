package gortsplib

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

type Conn struct {
	nconn             net.Conn
	writeBuf          []byte
	session           string
	clientCseqEnabled bool
	clientCseq        int
	clientAuthProv    *AuthClientProvider
}

func NewConn(nconn net.Conn) *Conn {
	return &Conn{
		nconn:    nconn,
		writeBuf: make([]byte, 2048),
	}
}

func (c *Conn) NetConn() net.Conn {
	return c.nconn
}

func (c *Conn) SetSession(v string) {
	c.session = v
}

func (c *Conn) ClientEnableCseq() {
	c.clientCseqEnabled = true
}

func (c *Conn) ClientSetCredentials(user string, pass string, realm string, nonce string) {
	c.clientAuthProv = NewAuthClientProvider(user, pass, realm, nonce)
}

func (c *Conn) ReadRequest() (*Request, error) {
	return readRequest(c.nconn)
}

func (c *Conn) WriteRequest(req *Request) error {
	if c.session != "" {
		if req.Header == nil {
			req.Header = make(Header)
		}
		req.Header["Session"] = []string{c.session}
	}
	if c.clientCseqEnabled {
		if req.Header == nil {
			req.Header = make(Header)
		}
		c.clientCseq += 1
		req.Header["CSeq"] = []string{strconv.FormatInt(int64(c.clientCseq), 10)}
	}
	if c.clientAuthProv != nil {
		if req.Header == nil {
			req.Header = make(Header)
		}
		req.Header["Authorization"] = []string{c.clientAuthProv.generateHeader(req.Method, req.Url)}
	}
	return req.write(c.nconn)
}

func (c *Conn) ReadResponse() (*Response, error) {
	return readResponse(c.nconn)
}

func (c *Conn) WriteResponse(res *Response) error {
	return res.write(c.nconn)
}

func (c *Conn) ReadInterleavedFrame(buf []byte) (int, int, error) {
	var header [4]byte
	_, err := io.ReadFull(c.nconn, header[:])
	if err != nil {
		return 0, 0, err
	}

	// connection terminated
	if header[0] == 0x54 {
		return 0, 0, io.EOF
	}

	if header[0] != 0x24 {
		return 0, 0, fmt.Errorf("wrong magic byte (0x%.2x)", header[0])
	}

	framelen := binary.BigEndian.Uint16(header[2:])
	if int(framelen) > len(buf) {
		return 0, 0, fmt.Errorf("frame length greater than buffer length")
	}

	_, err = io.ReadFull(c.nconn, buf[:framelen])
	if err != nil {
		return 0, 0, err
	}

	return int(header[1]), int(framelen), nil
}

func (c *Conn) WriteInterleavedFrame(channel int, frame []byte) error {
	c.writeBuf[0] = 0x24
	c.writeBuf[1] = byte(channel)
	binary.BigEndian.PutUint16(c.writeBuf[2:], uint16(len(frame)))
	n := copy(c.writeBuf[4:], frame)

	_, err := c.nconn.Write(c.writeBuf[:4+n])
	if err != nil {
		return err
	}
	return nil
}
