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
	Size     int64
}

func (fs *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)

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

	return nil
}

type MessageGetFile struct {
	Key string
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		return fs.store.Read(key)
	}

	fmt.Printf("don't have file (%s) locally fetching from network\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	for _, peer := range fs.peers {
		fileBuf := new(bytes.Buffer)
		n, err := io.Copy(fileBuf, peer)
		if err != nil {
			fmt.Printf("error with sending: %v", err)
			return nil, err
		}
		fmt.Println("received bytes over the network: ", n)
		fmt.Println(fileBuf.String())
	}

	select {}

	return nil, nil
}

// 1. Store this file to disk
// 2. broadcast this file to all known peers in network
func (fs *FileServer) StoreData(key string, r io.Reader) error {
	var (
		payloadBuffer = new(bytes.Buffer)
		tee           = io.TeeReader(r, payloadBuffer)
	)

	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Filename: key,
			Size:     size,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	// TODO: (@gnana997) use a multiwriter here.
	for _, peer := range fs.peers {
		n, err := io.Copy(peer, payloadBuffer)
		if err != nil {
			fmt.Printf("error with sending: %v", err)
			return err
		}

		fmt.Println("received and written bytes to disk: ", n)
	}

	return nil
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
		fmt.Println("file server stopped due to error or user quit action")
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
				log.Printf("error with decoder: %v\n", err)
			}

			if err := fs.handleMessage(msg.From, &m); err != nil {
				log.Printf("error handling message: %s\n", err)
			}
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from net.Addr, msg *Message) error {
	switch m := msg.Payload.(type) {
	case MessageStoreFile:
		fmt.Printf("storing file: %s\n", m.Filename)
		fs.handleMessageStoreFile(from, m)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, m)
	}
	return nil
}

func (fs *FileServer) handleMessageGetFile(from net.Addr, msg MessageGetFile) error {
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("need to serve file (%s) but it does not exist on disk", msg.Key)
	}

	fmt.Printf("serving file (%s) over the network\n", msg.Key)
	r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in peer map", from.String())
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes to peer: %s\n", n, from.String())
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from net.Addr, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer not found in peer map")
	}
	fmt.Printf("peer: %+v\n", peer)

	// this io.Copyy is blocker here
	// as the connection will not always
	// send EOF for io to stop copying
	if _, err := fs.store.Write(msg.Filename, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}

	fmt.Printf("written %d bytes to disk: %s\n", msg.Size, msg.Filename)

	peer.Streamed()

	return nil
}

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
	gob.Register(MessageGetFile{})
	gob.Register(MessageStoreFile{})
}
