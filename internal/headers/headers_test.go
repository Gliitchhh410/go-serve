package headers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders_SetAndGet(t *testing.T) {
	h := New()

	// Basic Set and Get
	h.Set([]byte("Content-Type"), []byte("text/plain"))
	val, ok := h.Get([]byte("content-type"))
	assert.True(t, ok)
	assert.Equal(t, []byte("text/plain"), val)

	//Case-Insensitivity
	val, ok = h.Get([]byte("CONTENT-TYPE"))
	assert.True(t, ok)
	assert.Equal(t, []byte("text/plain"), val)

	//Duplicate header appending
	h.Set([]byte("Accept"), []byte("text/plain"))
	h.Set([]byte("Accept"), []byte("application/json"))
	val, ok = h.Get([]byte("Accept"))
	assert.True(t, ok)
	assert.Equal(t, []byte("text/plain, application/json"), val)

	//Non-existent Header
	val, ok = h.Get([]byte("Non-Existent"))
	assert.False(t, ok)
	assert.Nil(t, val)

}

func TestHeaders_TooManyHeaders(t *testing.T) {
	h := New()

	for i := 0; i < MaxHeaders+1; i++ {
		err := h.Set([]byte(fmt.Sprintf("Header-%d", i)), []byte("value"))
		if i < MaxHeaders {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}

	assert.Equal(t, MaxHeaders, len(h.entries))
}

func TestHeaders_ObsoleteLineFolding(t *testing.T) {
	h := New()
	input := []byte("Header-Name: value\r\n folded value\r\n\r\n")
	_, _, err := h.Parse(input)
	assert.ErrorIs(t, err, ErrObsoleteLineFolding)
}

func TestHeaders_HeaderValueTooLarge(t *testing.T) {
	h := New()
	key := []byte("X-Long-Header")
	value := make([]byte, 1000)
	// 1000 * 4 + 2 * 3 = 4006 (OK)
	// 4006 + 2 + 1000 = 5008 (Too large)
	for i := 0; i < 4; i++ {
		err := h.Set(key, value)
		assert.NoError(t, err)
	}
	err := h.Set(key, value)
	assert.ErrorIs(t, err, ErrHeaderValueTooLarge)
}
