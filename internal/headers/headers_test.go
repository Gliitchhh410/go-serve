package headers

import (
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestHeaders_SetAndGet(t *testing.T){
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
