package headers

import (
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestHeaders_SetAndGet(t *testing.T){
	h := New()

	// Basic Set and Get
	h.Set("Content-Type", "text/plain")
	val, ok := h.Get("content-type")
	assert.True(t, ok)
	assert.Equal(t, "text/plain", val)

	//Case-Insensitivity
	val, ok = h.Get("content-type")
	assert.True(t, ok)
	
	assert.Equal(t, "text/plain", val)


	//Duplicate header appending
	h.Set("Accept", "text/plain")
	h.Set("Accept", "application/json")
	val, ok = h.Get("Accept")
	assert.True(t, ok)
	assert.Equal(t, "text/plain, application/json", val)


	//Non-existent Header 
	val, ok = h.Get("Non-Existent")
	assert.False(t, ok)
	assert.Equal(t, "", val)

}


func TestHeaders_RequiresNormalizedKeys(t *testing.T) {
	h := New()

	h.Set("Content-Type", "text/plain")

	val, ok := h.Get("content-type")
	assert.True(t, ok)
	assert.Equal(t, "text/plain", val)
}


func BenchmarkHeadersSetDuplicate(b *testing.B) {
	h := New()
	value := "192.168.1.1"
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {

		h.Set("x-forwarded-for", value)
	}
}
