// internal/request/request.go

// Package request implements a streaming HTTP/1.1 parser and state machine.

package request

import (
	"bytes"
	"errors"
	"io"
)

// Request represents a parsed HTTP request.
//
// It is built incrementally by the streaming parser and consists of:
//
//   1. Request line:
//      - Method  (e.g., GET, POST)
//      - Target  (e.g., /index.html)
//      - Version (e.g., HTTP/1.1)
//
//   2. Headers:
//      A collection of key-value pairs describing metadata.
//
//   3. Body:
//      Optional payload (used in POST, PUT, etc.).
//
//   4. State:
//      Tracks the current phase of the streaming parser
//      (e.g., parsing line, headers, or body).

type RequestLine struct {
	Method  string
	Target  string
	Version string
}

type Request struct {
	// Public fields intended for consumer
	Line *RequestLine
	Body []byte
	// Unexported fields representing the ownership boundary. Only the streaming parser may mutate these.
	state int // that's why it's s lower case
}

const (
	stateInit = iota
	stateHeaders
	stateBody
	stateDone
	stateError
)

const ( // Carriage Return + Line Feed
	crlf = "\r\n"
)

var (
	ErrMalformedRequest = errors.New("malformed HTTP request")
)

func NewRequest() *Request {
	return &Request{
		state: stateInit,
	}
}

func (r *Request) done() bool {
	if r.state == stateDone || r.state == stateError {
		return true
	}
	return false
}


func parseRequestLine(data []byte) (*RequestLine, int, error) {
	index := bytes.Index(data, []byte(crlf))
	if index == -1 {
		return nil, 0, nil
	}
	line := data[:index]

	consumed := index + len(crlf)

	parts := bytes.Split(line, []byte(" "))

	if len(parts) != 3 {
		return nil, 0, ErrMalformedRequest
	}

	method := string(parts[0])
	target := string(parts[1])
	version := string(parts[2])

	if len(method) == 0 || len(target) == 0 || len(version) == 0 {
		return nil, 0, ErrMalformedRequest
	}

	if version != "HTTP/1.1" {
		return nil, 0, ErrMalformedRequest
	}

	return &RequestLine{
		Method:  method,
		Target:  target,
		Version: version,
	}, consumed, nil

	
}

func (r *Request) parse(data []byte) (consumed int, err error){
	
	// _,_,_ := parseRequestLine(data)
	
	r.state = stateError
	return 0, errors.New("State Machine not implemented yet")
}



func RequestFromReader(r io.Reader) (*Request, error) {
	req := NewRequest()

	buf := make([]byte, 4096)
	bufLen := 0

	for !req.done() {
		n, err := r.Read(buf[bufLen:])

		if err == io.EOF {
			return nil, errors.New("Unexpected EOF")
		}
		if err != nil {
			return nil, err
		}
		bufLen += n

		consumed, err := req.parse(buf[:bufLen])

		if err != nil {
			return nil, err
		}

		if consumed < bufLen {
			copy(buf, buf[consumed:bufLen]) // shift the unconsumed bytes to the start of the buffer
			bufLen -= consumed
		}

	}
	return req, nil
}
