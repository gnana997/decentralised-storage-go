package main

import (
	"fmt"
	"log"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

func main() {

	tr := p2p.NewTCPTransport(":3000")

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("building decentralised storage system")

	select {}
}
