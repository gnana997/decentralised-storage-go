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

func main() {

	tcpOpts := p2p.TCPTransportOptions{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			rpc := <-tr.Consume()
			log.Printf("received rpc: %+v", rpc)
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("building decentralised storage system")

	select {}
}
