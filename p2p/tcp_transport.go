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
	// the underlying connection of the peer. Which in this case
	// is a net.TCPConn
	net.Conn

	// if we dial and retrieve a connection we are an outbound peer
	// if we accept and retrieve a connection we are an inbound peer
	outbound bool

	Wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (tp *TCPPeer) Send(payload []byte) error {
	if tp.Conn == nil {
		return errors.New("peer is closed")
	}

	_, err := tp.Conn.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

func (tp *TCPPeer) Streamed() {
	tp.Wg.Done()
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

// Connect implements the Transport interface,
// to connect to the peers over TCP.
func (t *TCPTransport) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
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

		go t.handleConn(conn, false)

	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		fmt.Printf("droppig peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, outbound)

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
		peer.Wg.Add(1)
		fmt.Printf("(%s) waiting till stream is done\n", t.ListenAddr)
		t.rpcch <- rpc
		peer.Wg.Wait()
		fmt.Println("stream is done")
		// write to the connection
		// close the connection
	}

}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	close(t.rpcch)
	return t.listener.Close()
}
