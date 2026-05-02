// internal/headers/headers.go

// Package headers provides a case-insensitive, multi-value HTTP header container.

package headers

import (
	"bytes"
	"errors"
)

const MaxHeaders = 100
var ErrTooManyHeaders = errors.New("too many headers")


type Header struct {
	Name  []byte
	Value []byte
}
type Headers struct {
	entries []Header
}

func New() *Headers {
	return &Headers{
		entries: make([]Header, 0, 32),
	}
}

// Set adds a Header. Keys are normalized to lowercase. Duplicate headers are concatenated with a comma.
func (h *Headers) Set(name []byte, value []byte) error {

	if len(h.entries) >= MaxHeaders {
		return ErrTooManyHeaders
	}
	for i := range h.entries {
		if bytes.EqualFold(h.entries[i].Name, name) {
			//Append ", " + value
			combined := make([]byte, len(h.entries[i].Value)+2+len(value))
			n := copy(combined, h.entries[i].Value)
			n += copy(combined[n:], ", ")
			copy(combined[n:], value)
			h.entries[i].Value = combined
			return nil
		}
	}
	
	h.entries = append(h.entries, Header{Name: name, Value: value})
	return nil
}

func (h *Headers) Get(name []byte) ([]byte, bool) {
	for i := range h.entries {
		if bytes.EqualFold(h.entries[i].Name, name) {
			return h.entries[i].Value, true
		}
	}
	return nil, false
}

func (h *Headers) ForEach(fn func(name, value []byte)) {
	for i := range h.entries {
		fn(h.entries[i].Name, h.entries[i].Value)
	}
}


func (h *Headers) Reset() {
	h.entries = h.entries[:0] // retain backing array for reuse
}

