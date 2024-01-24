package main

import (
	"bytes"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "encodedpathname"
	pathname := CASPathTransformFunc(key)
	expectedOriginalKey := "2d09e004d0c86cffa599f704573b8050a4e9e109"
	expectedPathName := "2d09e/004d0/c86cf/fa599/f7045/73b80/50a4e/9e109"
	if pathname.Path != expectedPathName {
		t.Errorf("expected %s, got %s", expectedPathName, pathname)
	}

	if pathname.Original != expectedOriginalKey {
		t.Errorf("expected %s, got %s", expectedPathName, pathname)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("hello world"))
	err := s.writeStream("tester", data)

	if err != nil {
		t.Error(err)
	}
}
