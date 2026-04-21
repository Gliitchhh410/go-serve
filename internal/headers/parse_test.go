package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders_ParsingHeaderLines(t *testing.T) {
	//Basic Valid Header
	name, value, err := parseHeader([]byte("Host: localhost"))
	assert.NoError(t, err)
	assert.Equal(t, "Host", name)
	assert.Equal(t, "localhost", value)

	//Multiple Colons
	name, value, err = parseHeader([]byte("Host: localhost:8080"))
	assert.NoError(t, err)
	assert.Equal(t, "Host", name)
	assert.Equal(t, "localhost:8080", value)

	//Missing Colon
	name, value, err = parseHeader([]byte("No Colon here, dude"))
	assert.Error(t, err)
	assert.Equal(t, "", name)
	assert.Equal(t, "", value)

	//Space on the left side
	name, value, err = parseHeader([]byte("Host : localhost"))
	assert.Error(t, err)
	assert.Equal(t, "", name)
	assert.Equal(t, "", value)

	//Invalid Token
	name, value, err = parseHeader([]byte("H@st: localhost"))
	assert.Error(t, err)
	assert.Equal(t, "", name)
	assert.Equal(t, "", value)

}
