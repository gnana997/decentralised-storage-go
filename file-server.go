package main

import (
	"fmt"

	"github.com/gnana997/decentralised-storage-go/p2p"
)

type FileServerOpts struct {
	RootFolder string

	Transport         p2p.Transport
	PathTransformFunc PathTransformFunc
}

type FileServer struct {
	FileServerOpts

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
	}
}

func (fs *FileServer) Stop() {
	close(fs.quitch)
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
			fmt.Println(msg)
		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	fs.loop()

	return nil
}

func (fs *FileServer) Close() error {
	return fs.store.Close()
}
