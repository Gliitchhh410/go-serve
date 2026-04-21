package headers

import (
	"bytes"
	"errors"
)

var ErrMalformedHeader = errors.New("malformed header")

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
