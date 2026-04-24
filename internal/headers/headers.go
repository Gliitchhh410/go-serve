// internal/headers/headers.go

// Package headers provides a case-insensitive, multi-value HTTP header container.

package headers

import "strings"

type Headers struct {
	m map[string]string
}

func New() *Headers {
	return &Headers{
		m: make(map[string]string),
	}
}

// Set adds a Header. Keys are normalized to lowercase. Duplicate headers are concatenated with a comma.
func (h *Headers) Set(name, value string) {
	key := strings.ToLower(name)

	if existing, ok := h.m[key]; ok {
		h.m[key] = existing + ", " + value
	} else {
		h.m[key] = value
	}
}

func (h *Headers) Get(name string) (string, bool) {
	key := strings.ToLower(name)
	value, ok := h.m[key]
	return value, ok
}

func (h *Headers) ForEach(fn func(name, value string)) {
	for name, value := range h.m {
		fn(name, value)
	}
}


