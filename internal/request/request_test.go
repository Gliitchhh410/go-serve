package request

import (
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
	
}
