package p2p

import "net"

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// Message represents any arbitrary data
// that is being sent over the transport
// between two nodes in the network.
type RPC struct {
	From    net.Addr
	Payload []byte
	Stream  bool
}
