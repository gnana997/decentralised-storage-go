package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddr := ":3000"
	tr := NewTCPTransport(listenAddr)

	assert.Equal(t, tr.listenAddr, listenAddr)

	assert.Nil(t, tr.ListenAndAccept())
}
