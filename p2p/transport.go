package p2p

import "net"

// Peer is an interface that represents the
// remote nodes in the network.
type Peer interface {
	net.Conn
	Streamed()
	Send([]byte) error
}

// Transport is anything that handles the communication
// between peers in the network. This can be of
// form (TCP, UDP, websocket, etc)
type Transport interface {
	Connect(addr string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
