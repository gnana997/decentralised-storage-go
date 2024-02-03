package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

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
		RootFolder:     "network_" + strings.Replace(listenAddr, ":", "", -1),
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

	time.Sleep(2 * time.Second)
	go s2.Start()
	time.Sleep(2 * time.Second)

	// for i := 0; i < 10; i++ {
	// 	data := bytes.NewReader([]byte("This is a very Large file"))
	// 	s2.StoreData(fmt.Sprintf("foo_%d", i), data)
	// 	time.Sleep(5 * time.Millisecond)
	// }

	data := bytes.NewReader([]byte("This is a very Large file"))
	s2.StoreData("foo", data)

	time.Sleep(2 * time.Second)

	r, err := s2.Get("foo")
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))
	select {}
}
