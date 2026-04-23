package headers

import (
	"bytes"
	"errors"
)

var ErrMalformedHeader = errors.New("malformed header")
const crlf = "\r\n"

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
	parts := bytes.SplitN(line, []byte(":"), 2)

	if len(parts) < 2 {
		return "", "", ErrMalformedHeader
	}

	name = string(parts[0])

	if len(name) < 1 || name[len(name)-1] == ' ' {
		return "", "", ErrMalformedHeader
	}

	for i := 0; i < len(name); i++ {
		if !isValidToken(name[i]) {
			return "", "", ErrMalformedHeader
		}
	}

	value = string(bytes.TrimSpace(parts[1]))

	return name, value, nil

}

func (h *Headers) Parse(data []byte) (consumed int, done bool, err error){
	for {
		index := bytes.Index(data[consumed:], []byte(crlf))

		if index == -1 {
			return consumed, false, nil
		}

		segmentLength := index + len(crlf)

		line := data[consumed: consumed+index]
		
		
		if len(line) == 0 {
			consumed += segmentLength
			return consumed, true, nil
		}
		name, value, err := parseHeader(line)
		
		if err != nil {
			return consumed, false, err
		}

		h.Set(name, value)
		consumed += segmentLength

	}
}

