package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return make(Headers)
}

var crlf = []byte("\r\n")

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, crlf)
	if idx == -1 {
		return 0, false, nil
	}
	if idx == 0 {
		return 2, true, nil
	}

	parts := bytes.SplitN(data[:idx], []byte(":"), 2)
	if len(parts) != 2 {
		return 0, false, fmt.Errorf("malformed header: %q", data[:idx])
	}

	if isInvalid(parts[0]) {
		return 0, false, fmt.Errorf("malformed header: %q", parts[0])
	}

	fieldName := bytes.TrimSpace(parts[0])
	fieldValue := bytes.TrimSpace(parts[1])

	h.Set(string(fieldName), string(fieldValue))

	return idx + 2, false, nil
}

func (h Headers) Set(key, value string) {
	key = strings.ToLower(key)
	if _, exist := h[key]; exist {
		h[key] = strings.Join(
			[]string{h[key], value},
			", ",
		)
		return
	}
	h[key] = value
}

func (h Headers) Get(key string) (string, bool) {
	key = strings.ToLower(key)
	v, ok := h[key]
	return v, ok
}

func (h Headers) Override(key, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

func (h Headers) Remove(key string) {
	key = strings.ToLower(key)
	delete(h, key)
}

// [RFC9910](https://datatracker.ietf.org/doc/html/rfc9110)
// Check for invalid whitespaces in the fieldName is covered with this check too
func isInvalid(fieldName []byte) bool {
	if len(fieldName) == 0 {
		return true
	}
	for _, c := range fieldName {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}

		switch c {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		}

		return true
	}
	return false
}
