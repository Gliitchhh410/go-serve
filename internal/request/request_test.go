package request

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type chunkReader struct {
	data      []byte
	chunkSize int
	index     int
}

func (c *chunkReader) Read(p []byte) (n int, err error) {
	if c.index >= len(c.data) {
		return 0, io.EOF
	}
	remaining := len(c.data) - c.index
	toBeTaken := min(len(p), c.chunkSize, remaining)
	copy(p, c.data[c.index:c.index+toBeTaken])
	c.index += toBeTaken
	return toBeTaken, nil
}

func TestRequestFromReader_RequestLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		chunkSize   int
		wantErr     bool
		wantMethod  string
		wantTarget  string
		wantVersion string
	}{
		{
			name:        "Valid Basic Request",
			input:       "GET / HTTP/1.1\r\n\r\n",
			chunkSize:   100,
			wantErr:     false,
			wantMethod:  "GET",
			wantTarget:  "/",
			wantVersion: "HTTP/1.1",
		},
		{
			name:      "Malformed: Broken Version Token",
			input:     "GET / /HTTP/1.1\r\n\r\n",
			chunkSize: 100,
			wantErr:   true,
		},
		{
			name:        "Valid Target Path",
			input:       "POST /coffee HTTP/1.1\r\n\r\n",
			chunkSize:   100,
			wantErr:     false,
			wantMethod:  "POST",
			wantTarget:  "/coffee",
			wantVersion: "HTTP/1.1",
		},
		{
			name:        "Fragmented Valid Request",
			input:       "GET /fragmented HTTP/1.1\r\n\r\n",
			chunkSize:   2, // stress test
			wantErr:     false,
			wantMethod:  "GET",
			wantTarget:  "/fragmented",
			wantVersion: "HTTP/1.1",
		},
		{
			name:      "Malformed: Missing Token",
			input:     "GET HTTP/1.1\r\n\r\n",
			chunkSize: 100,
			wantErr:   true,
		},
		{
			name:      "Malformed: Extra Token",
			input:     "GET / extra HTTP/1.1\r\n\r\n",
			chunkSize: 100,
			wantErr:   true,
		},
		{
			name:      "Malformed: Invalid Version",
			input:     "GET / HTTP/1.0\r\n\r\n",
			chunkSize: 100,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &chunkReader{
				data:      []byte(tt.input),
				chunkSize: tt.chunkSize,
			}

			req, err := RequestFromReader(cr)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, req)
			if req != nil {
				assert.Equal(t, tt.wantMethod, req.Line.Method)
				assert.Equal(t, tt.wantTarget, req.Line.Target)
				assert.Equal(t, tt.wantVersion, req.Line.Version)
			}
		})
	}
}

func TestRequestFromReader_Headers(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		chunkSize   int
		wantErr     bool
		wantHeaders map[string]string
	}{
		{
			name:      "Valid Headers",
			input:     "GET / HTTP/1.1\r\nHost: a\r\nAccept: b\r\n\r\n",
			chunkSize: 2,
			wantErr:   false,
			wantHeaders: map[string]string{
				"host":   "a",
				"accept": "b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &chunkReader{
				data:      []byte(tt.input),
				chunkSize: tt.chunkSize,
			}

			req, err := RequestFromReader(cr)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, req)
			if req != nil {
				for k, v := range tt.wantHeaders {
					val, ok := req.Headers.Get(k)
					assert.True(t, ok)
					assert.Equal(t, v, val)
				}
			}
		})
	}
}

func TestRequestFromReader_Body(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		chunkSize int
		wantErr   bool
		wantBody  string
	}{
		{
			name:      "Valid POST with Body",
			input:     "POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello",
			chunkSize: 100,
			wantErr:   false,
			wantBody:  "hello",
		},
		{
			name:      "Fragmented Body (chunkSize: 1)",
			input:     "POST / HTTP/1.1\r\nContent-Length: 4\r\n\r\ntest",
			chunkSize: 1,
			wantErr:   false,
			wantBody:  "test",
		},
		{
			name:      "Zero Body Default",
			input:     "GET / HTTP/1.1\r\n\r\n",
			chunkSize: 100,
			wantErr:   false,
			wantBody:  "",
		},
		{
			name:      "Invalid Content-Length",
			input:     "POST / HTTP/1.1\r\nContent-Length: abc\r\n\r\n",
			chunkSize: 100,
			wantErr:   true,
			wantBody:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &chunkReader{
				data:      []byte(tt.input),
				chunkSize: tt.chunkSize,
			}

			req, err := RequestFromReader(cr)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, req)
			if req != nil {
				assert.Equal(t, string(req.Body), tt.wantBody)
			}
		})
	}
}

func TestRequest_ErrorStateTrap(t *testing.T) {
	req := NewRequest()
	consumed, err := req.parse([]byte("GET / / HTTP/1.1\r\n\r\n")) // Malformed Extra Token '/'
	assert.Error(t, err)
	assert.Equal(t, stateError, req.state)

	consumed, err = req.parse([]byte("Valid subsequent data"))
	assert.NoError(t, err)
	assert.Equal(t, stateError, req.state)
	assert.Equal(t, 0, consumed)

}

func TestRequest_StateTransitions(t *testing.T) {
	req := NewRequest()
	assert.Equal(t, stateInit, req.state)

	consumed, err := req.parse([]byte("GET / HTTP/1.1\r\n"))
	assert.NoError(t, err)
	assert.Equal(t, 16, consumed)
	assert.Equal(t, stateHeaders, req.state)

	consumed, err = req.parse([]byte("Content-Length: 4\r\n\r\n"))
	assert.NoError(t, err)
	assert.Equal(t, 21, consumed)
	assert.Equal(t, stateBody, req.state)

	consumed, err = req.parse([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, consumed)
	assert.Equal(t, stateDone, req.state)

}

func BenchmarkRequestFromReader(b *testing.B) {
	raw := []byte("GET / HTTP/1.1\r\nHost: localhost\r\nUser-Agent: bench\r\n\r\n")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := RequestFromReader(bytes.NewReader(raw))
		if err != nil {
			b.Fatal(err)
		}
	}
}
