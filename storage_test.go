package main

import (
	"bytes"
	"fmt"
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
	s := newStore()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("Trip%d", i)
		data := bytes.NewReader([]byte("dude chill!!!"))

		if err := s.writeStream(key, data); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Error("expected true, got false")
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

		if err := s.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok {
			t.Error("expected false, got true")
		}
	}
}

func TestDeleteKey(t *testing.T) {
	s := newStore()
	defer teardown(t, s)

	key := "TripTomorrow"
	data := bytes.NewReader([]byte("dude chill!!!"))

	if err := s.writeStream(key, data); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
