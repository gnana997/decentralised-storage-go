package main

import (
	"fmt"
	"log"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

func main() {

	tcpOpts := p2p.TCPTransportOptions{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("building decentralised storage system")

	select {}
}
