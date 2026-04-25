// internal/request/request.go

// Package request implements a streaming HTTP/1.1 parser and state machine.

package request

import (
	"bytes"
	"errors"
	"httpServer/internal/headers"
	"io"
	"strconv"
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
//
// 	State transition graph:
//  Init -> Headers (on CRLF)
// 	Headers -> Body (on CRLF CRLF)
// 	Body -> Done (on Content-Length reached)
//

type RequestLine struct {
	Method  string
	Target  string
	Version string
}

type Request struct {
	// Public fields intended for consumer
	Line          *RequestLine
	Headers       *headers.Headers
	Body          []byte
	contentLength int
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

var ( // Carriage Return + Line Feed
	crlfBytes = []byte("\r\n")
	colonByte = byte(':')
)

var (
	ErrMalformedRequest = errors.New("malformed HTTP request")
)

func NewRequest() *Request {
	return &Request{
		Headers: headers.New(),
		state:   stateInit,
	}
}

func (r *Request) done() bool {
	if r.state == stateDone || r.state == stateError {
		return true
	}
	return false
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	index := bytes.Index(data, crlfBytes)
	if index == -1 {
		return nil, 0, nil
	}
	line := data[:index]

	consumed := index + len(crlfBytes)

	// parts := bytes.Split(line, []byte(" "))

	s1 := bytes.IndexByte(line, ' ')
	if s1 == -1 {
		return nil, 0, ErrMalformedRequest
	}

	s2 := bytes.IndexByte(line[s1+1:], ' ')
	if s2 == -1 {
		return nil, 0, ErrMalformedRequest
	}

	s2 += s1 + 1
	
	

	// if len(parts) != 3 {
	// 	return nil, 0, ErrMalformedRequest
	// }

	method := string(line[:s1])
	target := string(line[s1+1:s2])
	version := string(line[s2+1:])
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

// parse strictly requires CRLF line endings. It returns the exact number of consumed bytes, leaving unparsed data in the buffer. Body parsing strictly depends on the Content-Length header.
func (r *Request) parse(data []byte) (consumed int, err error) {
	consumed = 0
	for {

		if r.done() {
			return consumed, nil
		}
		switch r.state {
		case stateInit:
			line, n, err := parseRequestLine(data[consumed:])
			if err != nil{
				r.state = stateError
				return consumed + n, err
			}

			if line == nil {
				return consumed, nil
			}
			r.Line = line
			consumed += n
			r.state = stateHeaders

		case stateHeaders:
			n, done, err := r.Headers.Parse(data[consumed:])

			if err != nil {
				r.state = stateError
				return consumed + n, err
			}

			consumed += n
			if done {
				r.state = stateBody
			} else {
				return consumed, nil
			}

		case stateBody:
			if r.Body == nil {
				length, err := r.getContentLength()
				if err != nil {
					r.state = stateError
					return consumed, err
				}
				r.contentLength = length

				if length == 0 {
					r.Body = []byte{}
					r.state = stateDone
					continue
				}
				r.Body = make([]byte, 0, length)
			}

			remaining := r.contentLength - len(r.Body)
			available := len(data) - consumed
			toBeTaken := available
			if toBeTaken > remaining {
				toBeTaken = remaining
			}

			r.Body = append(r.Body, data[consumed:consumed+toBeTaken]...)
			consumed += toBeTaken

			if len(r.Body) == r.contentLength {
				r.state = stateDone
			}
			return consumed, nil
		default:
			return consumed, errors.New("Invalid parsing state")
		}

	}
	return consumed, nil

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
		if err != nil && err != io.EOF {
			return nil, err
		}
		bufLen += n

		consumed, err := req.parse(buf[:bufLen])

		if err != nil {
			return nil, err
		}

		if consumed > 0 {
			if consumed == bufLen {
				bufLen = 0
			} else {
				copy(buf, buf[consumed:bufLen])
				bufLen -= consumed
			}

		}

	}
	return req, nil
}

func (r *Request) getContentLength() (int, error) {
	val, ok := r.Headers.Get("content-length")
	if !ok {
		return 0, nil
	}

	contentLength, err := strconv.Atoi(val)
	if err != nil || contentLength < 0 {
		return 0, ErrMalformedRequest
	}

	return contentLength, nil

}
