package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStatusLine(t *testing.T) {
	// Test: Success Status
	statusLine := getStatusLine(StatusCodeSuccess)
	assert.Equal(t, "HTTP/1.1 200 OK\r\n", string(statusLine))

	// Test: Bad Request Status
	statusLine = getStatusLine(StatusCodeBadRequest)
	assert.Equal(t, "HTTP/1.1 400 Bad Request\r\n", string(statusLine))

	// Test: Internal Server Error Status
	statusLine = getStatusLine(StatusCodeInternalServerError)
	assert.Equal(t, "HTTP/1.1 500 Internal Server Error\r\n", string(statusLine))

	// Test: Unknown Status
	statusLine = getStatusLine(StatusCode(599))
	assert.Equal(t, "HTTP/1.1 599 \r\n", string(statusLine))
}
