package p2p

// Peer is an interface that represents the
// remote nodes in the network.
type Peer interface {
}

// Transport is anything that handles the communication
// between peers in the network. This can be of
// form (TCP, UDP, websocket, etc)
type Transport interface {
	ListenAndAccept() error
}
