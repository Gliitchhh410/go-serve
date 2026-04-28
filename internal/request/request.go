// internal/request/request.go

// Package request implements a streaming HTTP/1.1 parser and state machine.

package request

import (
	"bytes"
	"errors"
	"httpServer/internal/headers"
	"io"
	"strconv"
	"strings"
	"sync"
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

type RequestTarget struct {
	Path     string
	RawQuery string
}
type RequestLine struct {
	Method  string
	Target  RequestTarget
	Version string
}

type Request struct {
	// Public fields intended for consumer
	Line          *RequestLine
	Headers       *headers.Headers
	Body          []byte
	contentLength int
	state         int
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
)

var (
	ErrMalformedRequest = errors.New("malformed HTTP request")
	ErrInvalidState     = errors.New("invalid parsing state")
	ErrUnexpectedEOF    = errors.New("unexpected EOF")
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrInvalidTarget    = errors.New("invalid request target")
)

var allowedMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
}

var requestPool = sync.Pool{
	New: func() any {
		return NewRequest()
	},
}

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

func parseRequestTarget(raw string) (RequestTarget, error) {
	if len(raw) == 0 || raw[0] != '/' {
		return RequestTarget{}, ErrInvalidTarget
	}

	index := strings.IndexByte(raw, '?')

	if index == -1 {
		return RequestTarget{Path: raw, RawQuery: ""}, nil
	}

	return RequestTarget{
		Path:     raw[:index],
		RawQuery: raw[index+1:],
	}, nil
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	index := bytes.Index(data, crlfBytes)
	if index == -1 {
		return nil, 0, nil
	}
	line := data[:index]

	consumed := index + len(crlfBytes)

	s1 := bytes.IndexByte(line, ' ')
	if s1 == -1 {
		return nil, 0, ErrMalformedRequest
	}

	s2 := bytes.IndexByte(line[s1+1:], ' ')
	if s2 == -1 {
		return nil, 0, ErrMalformedRequest
	}

	s2 += s1 + 1

	method := string(line[:s1])
	targetRaw := string(line[s1+1 : s2])
	version := string(line[s2+1:])
	if len(method) == 0 || len(targetRaw) == 0 || len(version) == 0 {
		return nil, 0, ErrMalformedRequest
	}
	if version != "HTTP/1.1" {
		return nil, 0, ErrMalformedRequest
	}

	parsedTarget, err := parseRequestTarget(targetRaw)
	if err != nil {
		return nil, 0, err
	}


	return &RequestLine{
		Method:  method,
		Target:  parsedTarget,
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
			line, bytesParsed, err := parseRequestLine(data[consumed:])
			if err != nil {
				r.transitionTo(stateError)
				return consumed + bytesParsed, err
			}
			if line == nil {
				return consumed, nil
			}
			if err := validateMethod(line.Method); err != nil {
				r.transitionTo(stateError)
				return consumed + bytesParsed, err
			}

			r.Line = line
			consumed += bytesParsed
			r.transitionTo(stateHeaders)

		case stateHeaders:
			bytesParsed, done, err := r.Headers.Parse(data[consumed:])
			if err != nil {
				r.transitionTo(stateError)
				return consumed + bytesParsed, err
			}

			consumed += bytesParsed

			if done {
				// --- FIX START ---

				length, err := r.getContentLength()
				if err != nil {
					r.transitionTo(stateError)
					return consumed, err
				}

				r.contentLength = length

				if length == 0 {
					r.transitionTo(stateDone)
					continue
				}

				// Ensure capacity (reuse pooled slice if possible)
				if cap(r.Body) < length {
					r.Body = make([]byte, 0, length)
				} else {
					r.Body = r.Body[:0]
				}

				r.transitionTo(stateBody)

				// --- FIX END ---
			} else {
				return consumed, nil
			}

		case stateBody:
			// --- FIX: removed `if r.Body == nil` block entirely ---

			remaining := r.contentLength - len(r.Body)
			available := len(data) - consumed

			toBeTaken := available
			if toBeTaken > remaining {
				toBeTaken = remaining
			}

			r.Body = append(r.Body, data[consumed:consumed+toBeTaken]...)
			consumed += toBeTaken

			if len(r.Body) == r.contentLength {
				r.transitionTo(stateDone)
			}

			return consumed, nil

		default:
			return consumed, ErrInvalidState
		}
	}
}
func RequestFromReader(r io.Reader) (*Request, error) {
	req := AcquireRequest()

	readBuf := make([]byte, 4096)
	bufferedBytes := 0

	for !req.done() {
		bytesRead, err := r.Read(readBuf[bufferedBytes:])

		if err == io.EOF {
			ReleaseRequest(req)

			return nil, ErrUnexpectedEOF
		}
		if err != nil && err != io.EOF {
			ReleaseRequest(req)

			return nil, err
		}
		bufferedBytes += bytesRead

		consumed, err := req.parse(readBuf[:bufferedBytes])

		if err != nil {
			ReleaseRequest(req)
			return nil, err
		}

		if consumed > 0 {
			if consumed == bufferedBytes {
				bufferedBytes = 0
			} else {
				copy(readBuf, readBuf[consumed:bufferedBytes])
				bufferedBytes -= consumed
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

func (r *Request) transitionTo(nextState int) {
	r.state = nextState
}

func AcquireRequest() *Request {
	return requestPool.Get().(*Request)
}

func (r *Request) Reset() {
	r.Line = nil
	r.Headers.Reset()
	r.Body = r.Body[:0]
	r.contentLength = 0
	r.transitionTo(stateInit)
}

func ReleaseRequest(r *Request) {
	if r == nil {
		return
	}
	r.Reset()
	requestPool.Put(r)
}
func validateMethod(method string) error {
	_, ok := allowedMethods[method]
	if !ok {
		return ErrMethodNotAllowed
	}
	return nil
}
