package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultHeaders(t *testing.T) {
	// Test: Standard Default Headers
	h := GetDefaultHeaders(42)

	assert.Equal(t, "42", h["content-length"])
	assert.Equal(t, "close", h["connection"])
	assert.Equal(t, "text/plain", h["content-type"])
	assert.Len(t, h, 3)

	// Test: Zero Content Length
	h = GetDefaultHeaders(0)
	assert.Equal(t, "0", h["content-length"])
	assert.Equal(t, "close", h["connection"])
	assert.Equal(t, "text/plain", h["content-type"])
	assert.Len(t, h, 3)
}
