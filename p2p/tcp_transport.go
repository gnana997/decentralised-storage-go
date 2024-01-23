package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP connection.
type TCPPeer struct {
	// the underlying connection of the peer
	conn net.Conn

	// if we dial and retrieve a connection we are an outbound peer
	// if we accept and retrieve a connection we are an inbound peer
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddr string
	listener   net.Listener
	shakeHands HandshakeFunc
	decoder    Decoder

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddr: listenAddr,
		shakeHands: NOPHandshakeFunc,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %v\n", err)
			continue
		}

		go t.handleConn(conn)

	}
}

type Msg struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)
	fmt.Printf("handling connection %+v\n", peer)

	if err := t.shakeHands(peer); err != nil {
		fmt.Printf("handshake error: %v\n", err)
		return
	}

	lenDecodeError := 0
	//Read Loop
	msg := &Msg{}
	for {
		// read from the connection
		if err := t.decoder.Decode(conn, msg); err != nil {
			lenDecodeError++
			if lenDecodeError == 5 {
				fmt.Printf("dropping connection due to multiple decode errors: %+v\n", peer)
				return
			}
			fmt.Printf("decode error: %v\n", err)
			continue
		}
		// write to the connection
		// close the connection
	}

}
