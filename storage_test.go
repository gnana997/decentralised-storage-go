package main

import (
	"bytes"
	"io"
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

	if pathname.Filename != expectedOriginalKey {
		t.Errorf("expected %s, got %s", expectedPathName, pathname)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "TripTomorrow"
	data := bytes.NewReader([]byte("dude chill!!!"))

	if err := s.writeStream(key, data); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	if string(b) != "dude chill!!!" {
		t.Errorf("expected %s, got %s", "dude chill!!!", string(b))
	}
}
