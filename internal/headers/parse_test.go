package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders_ParsingHeaderLines(t *testing.T) {
	//Basic Valid Header
	name, value, err := parseHeader([]byte("Host: localhost"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("Host"), name)
	assert.Equal(t, []byte("localhost"), value)

	//Multiple Colons
	name, value, err = parseHeader([]byte("Host: localhost:8080"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("Host"), name)
	assert.Equal(t, []byte("localhost:8080"), value)

	//Missing Colon
	name, value, err = parseHeader([]byte("No Colon here, dude"))
	assert.Error(t, err)
	assert.Nil(t, name)
	assert.Nil(t, value)

	//Space on the left side
	name, value, err = parseHeader([]byte("Host : localhost"))
	assert.Error(t, err)
	assert.Nil(t, name)
	assert.Nil(t, value)

	//Invalid Token
	name, value, err = parseHeader([]byte("H@st: localhost"))
	assert.Error(t, err)
	assert.Nil(t, name)
	assert.Nil(t, value)

}

func TestHeaders_ParseBlock(t *testing.T) {
	// Complete Block
	h := New()
	consumed, done, err := h.Parse([]byte("Host: a\r\nAccept: b\r\n\r\n"))
	assert.NoError(t, err)
	assert.Equal(t, true, done)
	assert.Equal(t, 22, consumed)

	// Use Get() instead of h.m access
	val, ok := h.Get([]byte("host"))
	assert.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = h.Get([]byte("accept"))
	assert.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	// Partial Block
	h = New()
	consumed, done, err = h.Parse([]byte("Host: a\r\nAccept: b\r\n"))
	assert.NoError(t, err)
	assert.Equal(t, false, done)
	assert.Equal(t, 20, consumed)

	val, ok = h.Get([]byte("host"))
	assert.True(t, ok)
	assert.Equal(t, []byte("a"), val)

	val, ok = h.Get([]byte("accept"))
	assert.True(t, ok)
	assert.Equal(t, []byte("b"), val)

	// Malformed Block
	h = New()
	_, _, err = h.Parse([]byte("Host: a\r\nBad Header\r\n\r\n"))
	assert.Error(t, err)
}
