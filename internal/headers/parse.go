package headers

import (
	"bytes"
	"errors"
	"strings"
)

var ErrMalformedHeader = errors.New("malformed header")
var (
	crlfBytes = []byte("\r\n")
	colonByte = byte(':') 
)

func isValidToken(b byte) bool {
	isValid := false

	if b >= 'a' && b <= 'z' {
		isValid = true
	}

	if b >= 'A' && b <= 'Z' {
		isValid = true
	}

	if b >= '0' && b <= '9' {
		isValid = true
	}

	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		isValid = true
	}
	return isValid
}

func parseHeader(line []byte) (name string, value string, err error) {
	// parts := bytes.SplitN(line, []byte(":"), 2)
	s1 := bytes.IndexByte(line, colonByte)
	if s1 == -1 {
		return "", "", ErrMalformedHeader
	}
	name = string(line[:s1])
	value = string(bytes.TrimSpace(line[s1+1:]))



	if len(name) < 1 || name[len(name)-1] == ' ' {
		return "", "", ErrMalformedHeader
	}

	for i := 0; i < len(name); i++ {
		if !isValidToken(name[i]) {
			return "", "", ErrMalformedHeader
		}
	}
	return name, value, nil

}

func (h *Headers) Parse(data []byte) (consumed int, done bool, err error){
	for {
		index := bytes.Index(data[consumed:], crlfBytes)

		if index == -1 {
			return consumed, false, nil
		}

		segmentLength := index + len(crlfBytes)

		line := data[consumed: consumed+index]
		
		
		if len(line) == 0 {
			consumed += segmentLength
			return consumed, true, nil
		}
		name, value, err := parseHeader(line)
		
		if err != nil {
			return consumed, false, err
		}
		key := strings.ToLower(name)
		h.Set(key, value)
		consumed += segmentLength

	}
}

