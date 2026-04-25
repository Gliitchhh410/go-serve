// internal/headers/headers.go

// Package headers provides a case-insensitive, multi-value HTTP header container.

package headers

import "strings"

type Headers struct {
	m map[string][]string
}

func New() *Headers {
	return &Headers{
		m: make(map[string][]string),
	}
}

// Set adds a Header. Keys are normalized to lowercase. Duplicate headers are concatenated with a comma.
func (h *Headers) Set(name, value string) {
	// Fast path: assume already lowercase
	key := name

	// Only lowercase if needed
	for i := 0; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			key = strings.ToLower(name)
			break
		}
	}

	h.m[key] = append(h.m[key], value)
}

func (h *Headers) Get(name string) (string, bool) {
	key := strings.ToLower(name)
	values, ok := h.m[key]
	if !ok || len(values) == 0 {
		return "", false
	}
	return strings.Join(values, ", "), true
}

func (h *Headers) ForEach(fn func(name, value string)) {
	for name, values := range h.m {
		fn(name, strings.Join(values, ", "))
	}
}
