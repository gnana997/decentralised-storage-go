package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

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
		peer.Send([]byte{p2p.IncomingMessage})
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
		_, file, err := fs.store.Read(key)
		return file, err
	}

	fmt.Printf("[%s] don't have file (%s) locally fetching from network\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	for _, peer := range fs.peers {

		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		n, err := fs.store.Write(key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}
		// fileBuf := new(bytes.Buffer)
		// n, err := io.CopyN(fileBuf, peer, 25)
		// if err != nil {
		// 	fmt.Printf("error with sending: %v", err)
		// 	return nil, err
		// }
		fmt.Printf("[%s] received bytes over the network: %d\n", fs.Transport.Addr(), n)

		peer.CloseStream()
	}

	_, file, err := fs.store.Read(key)
	return file, err
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

	time.Sleep(2 * time.Millisecond)

	// TODO: (@gnana997) use a multiwriter here.
	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := io.Copy(peer, payloadBuffer)
		if err != nil {
			fmt.Printf("[%s] error with sending: %v\n", fs.Transport.Addr(), err)
			return err
		}

		fmt.Printf("[%s] received and written bytes to disk: %d", fs.Transport.Addr(), n)
		fmt.Printf("[%s] recieved content: %s\n", fs.Transport.Addr(), payloadBuffer.String())
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

	log.Printf("[%s] FileServer handling new peer: %s", fs.Transport.Addr(), p.RemoteAddr())

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		fmt.Printf("[%s] file server stopped due to error or user quit action", fs.Transport.Addr())
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
				log.Printf("[%s] error with decoder: %v\n", fs.Transport.Addr(), err)
			}

			if err := fs.handleMessage(msg.From, &m); err != nil {
				log.Printf("[%s] error handling message: %s\n", fs.Transport.Addr(), err)
			}
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from net.Addr, msg *Message) error {
	switch m := msg.Payload.(type) {
	case MessageStoreFile:
		fmt.Printf("[%s] storing file: %s\n", fs.Transport.Addr(), m.Filename)
		fs.handleMessageStoreFile(from, m)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, m)
	}
	return nil
}

func (fs *FileServer) handleMessageGetFile(from net.Addr, msg MessageGetFile) error {
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("[%s] need to serve file (%s) but it does not exist on disk", fs.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", fs.Transport.Addr(), msg.Key)
	fileSize, r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer (%s) not found in peer map", fs.Transport.Addr(), from.String())
	}

	// First Send the IncomingStream byte to peer
	if err := peer.Send([]byte{p2p.IncomingStream}); err != nil {
		return err
	}

	// Then send the file size as an int64 over the network
	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to peer: %s\n", fs.Transport.Addr(), n, from.String())
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from net.Addr, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer not found in peer map", fs.Transport.Addr())
	}
	fmt.Printf("peer: %+v\n", peer)

	// this io.LimitReader is blocker here
	// as the connection will not always
	// send EOF for io to stop copying
	if _, err := fs.store.Write(msg.Filename, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk: %s\n", fs.Transport.Addr(), msg.Size, msg.Filename)

	peer.CloseStream()

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
