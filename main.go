package main

import (
	"fmt"
	"log"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Println("doing some logic on peer outside transport")
	peer.Close()
	return nil
}

func makeServer(listenAddr string, nodes ...string) *FileServer {

	tcpTransportOpts := p2p.TCPTransportOptions{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
	}
	tcptransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		RootFolder:     listenAddr + "_network",
		BootstrapNodes: nodes,

		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcptransport,
	}

	s := NewFileServer(fileServerOpts)

	tcptransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	s2.Start()
}
