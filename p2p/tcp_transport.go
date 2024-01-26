package p2p

import (
	"errors"
	"fmt"
	"log"
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

// Close implements the peer interface.
func (tp *TCPPeer) Close() error {
	return tp.conn.Close()
}

type TCPTransportOptions struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOptions
	listener net.Listener
	rpcch    chan RPC

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOptions) *TCPTransport {
	return &TCPTransport{
		TCPTransportOptions: opts,
		rpcch:               make(chan RPC),
	}
}

// consume implements the Transport interface,
// which will return a read only channel for reading
// the incoming messages recieved from other peers.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP transport listening on %s", t.ListenAddr)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept error: %v\n", err)
			continue
		}

		go t.handleConn(conn)

	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		fmt.Printf("droppig peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, true)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	lenDecodeError := 0
	//Read Loop
	rpc := RPC{}
	for {
		// read from the connection
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			lenDecodeError++
			if lenDecodeError == 5 {
				fmt.Printf("TCP dropping connection due to multiple decode errors: %+v\n", peer)
				return
			}
			fmt.Printf("decode error: %+v\n", err)
			continue
		}

		rpc.From = conn.RemoteAddr()
		t.rpcch <- rpc
		// write to the connection
		// close the connection
	}

}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	close(t.rpcch)
	return t.listener.Close()
}
