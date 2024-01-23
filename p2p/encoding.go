package p2p

import (
	"bytes"
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
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}

	rpc.Payload = buf.Bytes()[:n]

	return nil
}
