// internal/headers/headers.go

// Package headers provides a case-insensitive, multi-value HTTP header container.

package headers

import (
	"bytes"
)

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
func (h *Headers) Set(name []byte, value []byte) {
	for i := range h.entries {
		if bytes.EqualFold(h.entries[i].Name, name) {
			//Append ", " + value
			combined := make([]byte, len(h.entries[i].Value)+2+len(value))
			n := copy(combined, h.entries[i].Value)
			n += copy(combined[n:], ", ")
			copy(combined[n:], value)
			h.entries[i].Value = combined
			return
		}
	}
	h.entries = append(h.entries, Header{Name: name, Value: value})
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

func (h *Headers) Own() {
	for i := range h.entries {
		h.entries[i].Name = bytes.Clone(h.entries[i].Name)
		h.entries[i].Value = bytes.Clone(h.entries[i].Value)
	}
}
