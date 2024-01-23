package p2p

import "errors"

//ErrInvalidHandshake is returned if the handshake between
// the local and remote node could not be established.
var ErrInvalidHandShake = errors.New("invalid handshake")

type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error {
	return nil
}
