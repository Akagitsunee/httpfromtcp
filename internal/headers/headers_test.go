package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T) {
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)

	// Test: Valid single header with extra whitespace
	headers = NewHeaders()
	data = []byte("Host: localhost:42069       \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 30, n)
	assert.False(t, done)

	// Test: Valid 2 headers with existing headers
	headers = map[string]string{"host": "localhost:42069"}
	data = []byte("User-AgEnT: curl/7.81.0\r\nAccept: */*\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, "curl/7.81.0", headers["user-agent"])
	assert.Equal(t, 25, n)
	assert.False(t, done)

	// Test: Valid multiple values for same header with existing headers
	headers = map[string]string{"host": "localhost:42069", "set-person": "foo-bar"}
	data = []byte("Set-Person: fib-fab\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, "foo-bar, fib-fab", headers["set-person"])
	assert.Equal(t, 21, n)
	assert.False(t, done)

	// Test: Valid done
	headers = NewHeaders()
	data = []byte("\r\n a bunch of other stuff")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Empty(t, headers)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("       Host: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid character header
	headers = NewHeaders()
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
}

func TestHeadersMethodsNormalizeKeys(t *testing.T) {
	// Test: Set and Get Case Insensitive Header
	headers := NewHeaders()

	headers.Set("Content-Type", "text/plain")
	value, ok := headers.Get("content-type")
	require.True(t, ok)
	assert.Equal(t, "text/plain", value)

	// Test: Set Duplicate Case Insensitive Header
	headers.Set("CONTENT-TYPE", "text/html")
	value, ok = headers.Get("Content-Type")
	require.True(t, ok)
	assert.Equal(t, "text/plain, text/html", value)

	// Test: Override Case Insensitive Header
	headers.Override("content-TYPE", "application/json")
	value, ok = headers.Get("CONTENT-type")
	require.True(t, ok)
	assert.Equal(t, "application/json", value)

	// Test: Remove Case Insensitive Header
	headers.Remove("CONTENT-TYPE")
	_, ok = headers.Get("content-type")
	assert.False(t, ok)
}
