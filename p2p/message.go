package p2p

import "net"

// Message represents any arbitrary data
// that is being sent over the transport
// between two nodes in the network.
type RPC struct {
	From    net.Addr
	Payload []byte
}
