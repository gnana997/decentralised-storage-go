package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, rpc *RPC) error {
	return gob.NewDecoder(r).Decode(rpc)
}

type NOPDecoder struct{}

func (dec NOPDecoder) Decode(r io.Reader, rpc *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	// Incase of stream, we don't need to read the payload
	// sent over the network. We are just setting the Stream
	// flag to true, So we can handle in file-server.go
	stream := peekBuf[0] == IncomingStream
	if stream {
		rpc.Stream = true
		return nil
	}

	buf := make([]byte, 1028)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}

	rpc.Payload = buf[:n]

	return nil
}
