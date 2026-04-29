// internal/request/request.go

// Package request implements a streaming HTTP/1.1 parser and state machine.

package request

import (
	"bytes"
	"errors"
	"httpServer/internal/headers"
	"io"
	"path"
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
	Line               *RequestLine
	Host               string
	Headers            *headers.Headers
	Body               []byte
	contentLength      int
	transferEncoding   int
	state              int
	chunkSizeRemaining int
	parsingChunkHeader bool
}

const (
	stateInit = iota
	stateHeaders
	stateBody
	stateDone
	stateError
)

const (
	encodingIdentity = iota // Standard Content-Length parsing
	encodingChunked         // Chunked Parsing
)

var ( // Carriage Return + Line Feed
	crlfBytes = []byte("\r\n")
)

var (
	ErrMalformedRequest            = errors.New("malformed HTTP request")
	ErrInvalidState                = errors.New("invalid parsing state")
	ErrUnexpectedEOF               = errors.New("unexpected EOF")
	ErrMethodNotAllowed            = errors.New("method not allowed")
	ErrInvalidTarget               = errors.New("invalid request target")
	ErrMissingHost                 = errors.New("missing or invalid Host header")
	ErrUnsupportedTransferEncoding = errors.New("unsupported transfer encoding")
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
		Headers:            headers.New(),
		state:              stateInit,
		parsingChunkHeader: true,
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
		cleanPath := path.Clean(raw)
		return RequestTarget{Path: cleanPath, RawQuery: ""}, nil
	}

	return RequestTarget{
		Path:     path.Clean(raw[:index]),
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
				err := r.validateHost()

				if err != nil {
					r.transitionTo(stateError)
					return consumed, err
				}

				te, ok := r.Headers.Get("transfer-encoding")

				if ok {
					err = checkTransferEncoding(te)
					if err != nil {
						r.transitionTo(stateError)
						return consumed, err
					}
				}

				if ok && te == "chunked" {
					r.transferEncoding = encodingChunked
					r.transitionTo(stateBody)
					continue
				}

				r.transferEncoding = encodingIdentity
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

			} else {
				return consumed, nil
			}

		case stateBody:

			if r.transferEncoding == encodingChunked {
				bytesParsed, done, err := r.parseChunkBody(data[consumed:])
				consumed += bytesParsed
				if err != nil {
					r.transitionTo(stateError)
					return consumed, err
				}
				if done {
					r.transitionTo(stateDone)
				}
				return consumed, nil
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
	r.Host = ""
	r.transferEncoding = 0
	r.Headers.Reset()
	r.Body = r.Body[:0]
	r.contentLength = 0
	r.chunkSizeRemaining = 0
	r.parsingChunkHeader = true
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

func (r *Request) validateHost() error {
	host, ok := r.Headers.Get("host")

	if !ok || len(host) == 0 || strings.Contains(host, ",") {
		return ErrMissingHost
	}

	r.Host = host
	return nil

}

func (r *Request) parseChunkBody(data []byte) (consumed int, done bool, err error) {
	consumed = 0

	for consumed < len(data) {
		if r.parsingChunkHeader {
			index := bytes.Index(data[consumed:], crlfBytes)
			if index == -1 {
				return consumed, false, nil // wait for full chunk header
			}

			hexData := data[consumed : consumed+index]
			semiIdx := bytes.IndexByte(hexData, ';')
			if semiIdx != -1 {
				hexData = hexData[:semiIdx]
			}

			size, err := strconv.ParseInt(string(hexData), 16, 64)
			if err != nil {
				return consumed, false, ErrMalformedRequest
			}

			if size == 0 {
				// Final chunk: '0' + CRLF + final CRLF
				totalNeeded := index + 4
				if len(data[consumed:]) < totalNeeded {
					return consumed, false, nil // wait for final CRLF
				}

				if !bytes.Equal(data[consumed+index+2:consumed+totalNeeded], crlfBytes) {
					return consumed, false, ErrMalformedRequest
				}

				consumed += totalNeeded
				return consumed, true, nil
			}

			consumed += index + 2
			r.chunkSizeRemaining = int(size)
			r.parsingChunkHeader = false

		} else {
			// Phase 1: Consume Payload
			if r.chunkSizeRemaining > 0 {
				available := len(data) - consumed
				take := available
				if take > r.chunkSizeRemaining {
					take = r.chunkSizeRemaining
				}
				r.Body = append(r.Body, data[consumed:consumed+take]...)
				r.chunkSizeRemaining -= take
				consumed += take
			}

			// Phase 2: Consume trailing CRLF
			if r.chunkSizeRemaining == 0 {
				available := len(data) - consumed
				if available < 2 {
					return consumed, false, nil // wait for the trailing CRLF
				}
				if !bytes.Equal(data[consumed:consumed+2], crlfBytes) {
					return consumed, false, ErrMalformedRequest
				}
				consumed += 2
				r.parsingChunkHeader = true
			}
		}
	}

	return consumed, false, nil
}

// checkTransferEncoding validates Transfer-Encoding for this parser.
//
// The parser supports only "identity" and "chunked" transfer encodings.
// Any other Transfer-Encoding value returns ErrUnsupportedTransferEncoding.
func checkTransferEncoding(value string) error {
	if value != "identity" && value != "chunked" {
		return ErrUnsupportedTransferEncoding
	}
	return nil

}
