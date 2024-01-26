package main

import (
	"fmt"
	"time"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Println("doing some logic on peer outside transport")
	peer.Close()
	return nil
}

func main() {

	tcpTransportOpts := p2p.TCPTransportOptions{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
	}
	tcptransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		RootFolder:        "3000_network",
		PathTransformFunc: CASPathTransformFunc,

		Transport: tcptransport,
	}

	fs := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(time.Second * 3)
		fs.Stop()
	}()

	if err := fs.Start(); err != nil {
		fmt.Println(err)
		return
	}
}
