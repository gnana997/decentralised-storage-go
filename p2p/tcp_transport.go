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

type TCPTransportOptions struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOptions
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOptions) *TCPTransport {
	return &TCPTransport{
		TCPTransportOptions: opts,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
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

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)
	fmt.Printf("TCP handling connection %+v\n", peer)

	if err := t.HandshakeFunc(peer); err != nil {
		fmt.Printf("TCP handshake error: %v\n", err)
		conn.Close()
		return
	}

	lenDecodeError := 0
	//Read Loop
	rpc := &RPC{}
	for {
		// read from the connection
		if err := t.Decoder.Decode(conn, rpc); err != nil {
			lenDecodeError++
			if lenDecodeError == 5 {
				fmt.Printf("TCP dropping connection due to multiple decode errors: %+v\n", peer)
				return
			}
			fmt.Printf("decode error: %v\n", err)
			continue
		}

		rpc.From = conn.RemoteAddr()
		fmt.Printf("message: %+v\n", rpc)
		// write to the connection
		// close the connection
	}

}
