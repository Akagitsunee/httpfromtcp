package server

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/akagitusnee/httpfromtcp/internal/request"
	"github.com/akagitusnee/httpfromtcp/internal/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConn struct {
	reader *bytes.Reader
	writer bytes.Buffer
	closed bool
}

func newTestConn(input string) *testConn {
	return &testConn{
		reader: bytes.NewReader([]byte(input)),
	}
}

func (c *testConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *testConn) Write(p []byte) (int, error) {
	return c.writer.Write(p)
}

func (c *testConn) Close() error {
	c.closed = true
	return nil
}

func (c *testConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c *testConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c *testConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *testConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *testConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

type testAddr string

func (a testAddr) Network() string {
	return "test"
}

func (a testAddr) String() string {
	return string(a)
}

func TestServerHandleDispatchesParsedRequestToHandler(t *testing.T) {
	// Test: Valid Request Is Dispatched To Handler
	conn := newTestConn("GET /hello HTTP/1.1\r\nHost: example.test\r\n\r\n")

	handlerCalled := false
	var handlerErr error
	s := &Server{
		handler: func(w *response.Writer, req *request.Request) {
			handlerCalled = true
			assert.Equal(t, "GET", req.RequestLine.Method)
			assert.Equal(t, "/hello", req.RequestLine.RequestTarget)
			assert.Equal(t, "example.test", req.Headers["host"])

			body := []byte("handled")
			h := response.GetDefaultHeaders(len(body))
			if err := w.WriteStatusLine(response.StatusCodeSuccess); err != nil {
				handlerErr = err
				return
			}
			if err := w.WriteHeaders(h); err != nil {
				handlerErr = err
				return
			}
			if _, err := w.WriteBody(body); err != nil {
				handlerErr = err
				return
			}
		},
	}

	s.handle(conn)
	require.NoError(t, handlerErr)

	got := conn.writer.String()
	assert.True(t, handlerCalled)
	assert.True(t, strings.HasPrefix(got, "HTTP/1.1 200 OK\r\n"))
	assert.Contains(t, got, "content-length: 7\r\n")
	assert.True(t, strings.HasSuffix(got, "\r\n\r\nhandled"))
	assert.True(t, conn.closed)
}

func TestServerHandleWritesBadRequestWhenRequestParsingFails(t *testing.T) {
	// Test: Invalid Request Writes 400 Response
	conn := newTestConn("get /bad HTTP/1.1\r\nHost: example.test\r\n\r\n")

	handlerCalled := false
	s := &Server{
		handler: func(_ *response.Writer, _ *request.Request) {
			handlerCalled = true
		},
	}

	s.handle(conn)

	got := conn.writer.String()
	assert.False(t, handlerCalled)
	assert.True(t, strings.HasPrefix(got, "HTTP/1.1 400 Bad Request\r\n"))
	assert.Contains(t, got, "Error parsing request:")
	assert.Contains(t, got, "invalid method")
	assert.True(t, conn.closed)
}

func TestServerCloseWithNilListener(t *testing.T) {
	// Test: Close Without Listener
	var s Server

	require.NoError(t, s.Close())
	assert.True(t, s.closed.Load())
}
