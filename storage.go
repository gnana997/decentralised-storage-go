package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "dsg"

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
	// root is the folder name where
	// all the files will be stored
	Root              string
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Path:     key,
		Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
	fi, err := os.Stat(fullPathWithRoot)
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

	firstPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathFolder())
	return os.RemoveAll(firstPathWithRoot)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
	// if err != nil {
	// 	return 0, nil, err
	// }
	// defer f.Close()

	// buf := new(bytes.Buffer)
	// _, err = io.Copy(buf, f)
	// if err != nil {
	// 	return 0, nil, err
	// }

	// return n, buf, nil
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	pathKeyWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

	file, err := os.Open(pathKeyWithRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, err
		}
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Path)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return 0, err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, fullPath)
	f, err := os.Create(fullPathWithRoot)
	if err != nil {
		return 0, err
	}
	defer func() {
		log.Printf("closed file: %s", fullPathWithRoot)
		f.Close()
	}()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}

	log.Printf("written (%d) bytes to disk: %s", n, fullPathWithRoot)

	return n, nil
}

func (s *Store) Close() error {
	return os.RemoveAll(s.Root)
}
