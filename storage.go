package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i+1)*blocksize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Path:     strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

type PathKey struct {
	Path     string
	Filename string
}

func (p PathKey) FirstPathFolder() string {
	paths := strings.Split(p.Path, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Path, p.Filename)
}

type PathTransformFunc func(string) PathKey

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) string {
	return key
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fi, err := os.Stat(pathKey.FullPath())
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatal(err)
	}

	if !fi.IsDir() {
		return true
	}

	return false
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted from disk: %s", pathKey.Filename)
	}()

	return os.RemoveAll(pathKey.FirstPathFolder())
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, f)
	if err != nil {
		return nil, err
	}

	log.Printf("read (%d) bytes from disk: %s", n, key)

	return buf, nil
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)
	if err := os.MkdirAll(pathKey.Path, os.ModePerm); err != nil {
		return err
	}

	pathAndFilename := pathKey.FullPath()
	f, err := os.Create(pathAndFilename)
	if err != nil {
		return err
	}
	defer func() {
		log.Printf("closed file: %s", pathAndFilename)
		f.Close()
	}()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk: %s", n, pathAndFilename)

	return nil
}
