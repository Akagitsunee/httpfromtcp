package response

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/akagitusnee/httpfromtcp/internal/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingWriter struct {
	err error
}

func (fw failingWriter) Write(_ []byte) (int, error) {
	return 0, fw.err
}

func TestWriterWritesFixedLengthResponseInOrder(t *testing.T) {
	// Test: Standard Fixed-Length Response
	var buf bytes.Buffer
	w := NewWriter(&buf)

	require.NoError(t, w.WriteStatusLine(StatusCodeSuccess))

	h := headers.NewHeaders()
	h.Set("Content-Length", "5")
	h.Set("Content-Type", "text/plain")
	require.NoError(t, w.WriteHeaders(h))

	n, err := w.WriteBody([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	got := buf.String()
	assert.True(t, strings.HasPrefix(got, "HTTP/1.1 200 OK\r\n"))
	assert.Contains(t, got, "content-length: 5\r\n")
	assert.Contains(t, got, "content-type: text/plain\r\n")
	assert.Contains(t, got, "\r\n\r\nhello")
}

func TestWriterRejectsOutOfOrderWrites(t *testing.T) {
	// Test: Body Before Status Line
	var buf bytes.Buffer
	w := NewWriter(&buf)

	_, err := w.WriteBody([]byte("too soon"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write body in state")

	// Test: Headers Before Status Line
	buf = bytes.Buffer{}
	w = NewWriter(&buf)
	err = w.WriteHeaders(headers.NewHeaders())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write headers in state")

	// Test: Status Line Twice
	buf = bytes.Buffer{}
	w = NewWriter(&buf)
	require.NoError(t, w.WriteStatusLine(StatusCodeSuccess))

	err = w.WriteStatusLine(StatusCodeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write status line in state")

	// Test: Trailers Before Chunked Body Done
	buf = bytes.Buffer{}
	w = NewWriter(&buf)
	require.NoError(t, w.WriteStatusLine(StatusCodeSuccess))
	require.NoError(t, w.WriteHeaders(headers.NewHeaders()))

	err = w.WriteTrailers(headers.NewHeaders())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write body in state")
}

func TestWriterWritesChunkedBodyAndTrailers(t *testing.T) {
	// Test: Standard Chunked Response
	var buf bytes.Buffer
	w := NewWriter(&buf)

	require.NoError(t, w.WriteStatusLine(StatusCodeSuccess))

	h := headers.NewHeaders()
	h.Set("Transfer-Encoding", "chunked")
	require.NoError(t, w.WriteHeaders(h))

	n, err := w.WriteChunkedBody([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, len("5\r\nhello\r\n"), n)

	n, err = w.WriteChunkedBodyDone()
	require.NoError(t, err)
	assert.Equal(t, len("0\r\n"), n)

	trailers := headers.NewHeaders()
	trailers.Override("X-Content-Length", "5")
	trailers.Override("X-Content-SHA256", "abc123")
	require.NoError(t, w.WriteTrailers(trailers))

	got := buf.String()
	assert.Contains(t, got, "transfer-encoding: chunked\r\n")
	assert.Contains(t, got, "\r\n\r\n5\r\nhello\r\n0\r\n")
	assert.Contains(t, got, "x-content-length: 5\r\n")
	assert.Contains(t, got, "x-content-sha256: abc123\r\n")
	assert.True(t, strings.HasSuffix(got, "\r\n\r\n"))
}

func TestWriterRejectsChunkedWritesBeforeBodyState(t *testing.T) {
	// Test: Chunked Body Before Status Line
	var buf bytes.Buffer
	w := NewWriter(&buf)

	_, err := w.WriteChunkedBody([]byte("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write body in state")

	// Test: Chunked Body Done Before Status Line
	buf = bytes.Buffer{}
	w = NewWriter(&buf)
	_, err = w.WriteChunkedBodyDone()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot write body in state")
}

func TestWriterReturnsUnderlyingWriteErrors(t *testing.T) {
	// Test: Status Line Write Error
	writeErr := errors.New("write failed")
	w := NewWriter(failingWriter{err: writeErr})

	err := w.WriteStatusLine(StatusCodeSuccess)
	require.ErrorIs(t, err, writeErr)
}
