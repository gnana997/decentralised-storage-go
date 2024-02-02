package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

type FileServerOpts struct {
	RootFolder     string
	BootstrapNodes []string

	Transport         p2p.Transport
	PathTransformFunc PathTransformFunc
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[net.Addr]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	return &FileServer{
		FileServerOpts: opts,
		store: NewStore(StoreOpts{
			Root:              opts.RootFolder,
			PathTransformFunc: opts.PathTransformFunc,
		}),
		quitch: make(chan struct{}),
		peers:  make(map[net.Addr]p2p.Peer),
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Filename string
}

func (fs *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	// 1. Store this file to disk
	// 2. broadcast this file to all known peers in network
	// log.Println("Storing file", key)

	buf := new(bytes.Buffer)

	msg := Message{
		Payload: MessageStoreFile{
			Filename: key,
		},
	}

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		fmt.Printf("error with encoder: %v", err)
		return err
	}

	for _, peer := range fs.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			fmt.Printf("error with sending: %v", err)
			return err
		}
	}

	// time.Sleep(time.Second * 3)

	// payload := []byte("This Large File")
	// for _, peer := range fs.peers {
	// 	if err := peer.Send(payload); err != nil {
	// 		fmt.Printf("error with sending: %v", err)
	// 		return err
	// 	}
	// }

	return nil
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	// if err := fs.store.Write(key, tee); err != nil {
	// 	return err
	// }

	// // the reader is empty now

	// fmt.Println(buf.Bytes())

	// p := &DataMessage{
	// 	Key:   key,
	// 	Value: buf.Bytes(),
	// }

	// return fs.broadcast(&Message{
	// 	From:    "TODO",
	// 	Payload: p,
	// })
}

func (fs *FileServer) Stop() {
	close(fs.quitch)
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr()] = p

	log.Printf("FileServer handling new peer: %s", p.RemoteAddr())

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		if err := fs.Transport.Close(); err != nil {
			fmt.Println(err)
		}
		if err := fs.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	for {
		select {
		case msg := <-fs.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				fmt.Printf("error with decoder: %v", err)
				log.Fatal(err)
			}
			fmt.Printf("received message: %+v\n", m.Payload)

			peer, ok := fs.peers[msg.From]
			if !ok {
				panic("peer not found in peer map")
			}

			b := make([]byte, 1028)
			if _, err := peer.Read(b); err != nil {
				panic(err)
			}

			peer.Streamed()

			fmt.Printf("received message: %v\n", string(b))
			// if err := fs.handleMessage(&m); err != nil {
			// log.Printf("error handling message: %s", err)
			// }
		case <-fs.quitch:
			return
		}
	}
}

// func (fs *FileServer) handleMessage(msg *Message) error {
// 	switch m := msg.Payload.(type) {
// 	case *DataMessage:
// 		return fs.StoreData(m.Key, bytes.NewReader(m.Value))
// 	default:
// 		return fmt.Errorf("unknown message type: %T", msg.Payload)
// 	}
// }

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Printf("dialing %s\n", addr)
			if err := fs.Transport.Connect(addr); err != nil {
				log.Printf("dial error: %s", err)
			}
		}(addr)
	}
	return nil
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(fs.BootstrapNodes) != 0 {
		fs.bootstrapNetwork()
	}

	fs.loop()

	return nil
}

func (fs *FileServer) Close() error {
	return fs.store.Close()
}

func init() {
	gob.Register(MessageStoreFile{})
}
