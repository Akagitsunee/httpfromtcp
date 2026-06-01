# httpfromtcp

`httpfromtcp` is a small HTTP/1.1 implementation written in Go from the TCP layer up. It parses raw bytes from a `net.Conn`, turns them into request objects, writes HTTP response bytes directly to an `io.Writer`, and includes a demo server that exercises normal responses, error responses, static video serving, and chunked proxy responses with trailers.

The project is intentionally low-level. It does not use Go's `net/http` server machinery for inbound requests. The only use of `net/http` is in the demo proxy route, where the server makes an outbound request to `https://httpbin.org`.

## Contents

- [Goals](#goals)
- [Repository Layout](#repository-layout)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Demo Server](#demo-server)
- [Command-Line Programs](#command-line-programs)
- [Internal Packages](#internal-packages)
- [HTTP Behavior](#http-behavior)
- [Testing](#testing)
- [Linting And Formatting](#linting-and-formatting)
- [Development Notes](#development-notes)
- [Current Limitations](#current-limitations)

## Goals

This project is useful for learning and experimenting with:

- TCP listeners and connection handling with `net.Listen` and `net.Conn`
- Incremental parsing from an `io.Reader`
- HTTP/1.1 request-line parsing
- Header parsing and normalization
- Request body parsing with `Content-Length`
- HTTP response serialization
- Chunked transfer encoding
- HTTP trailers
- Basic concurrent connection handling
- The boundary between HTTP as bytes on a socket and higher-level web frameworks

It should be read as an educational implementation, not as a drop-in replacement for Go's standard `net/http` server.

## Repository Layout

```text
.
|-- assets/
|   `-- vim.mp4                    # Video file served by the demo /video route
|-- cmd/
|   |-- httpserver/
|   |   `-- main.go                # Demo HTTP server built on the internal server package
|   |-- tcplistener/
|   |   `-- main.go                # Raw TCP listener that parses and prints one HTTP request per connection
|   `-- udpsender/
|       `-- main.go                # Small interactive UDP sender utility
|-- internal/
|   |-- headers/
|   |   |-- headers.go             # Header map, parser, validation, and helper methods
|   |   `-- headers_test.go
|   |-- request/
|   |   |-- request.go             # Incremental HTTP request parser
|   |   `-- request_test.go
|   |-- response/
|   |   |-- headers.go             # Default response headers
|   |   |-- status_line.go         # HTTP status line serialization
|   |   `-- writer.go              # Stateful response writer
|   `-- server/
|       |-- handler.go             # Handler type definition
|       `-- server.go              # TCP server loop and request dispatch
|-- go.mod
|-- go.sum
|-- golangci.yml
`-- messages.txt
```

## Requirements

- Go 1.26 or newer, matching the `go.mod` file
- A shell with access to local TCP port `42069`
- Internet access only if you want to use the `/httpbin/*` proxy route

The project currently depends on `github.com/stretchr/testify` for tests. The remaining listed modules in `go.sum` are transitive dependencies used by the test tooling.

## Quick Start

Run the demo HTTP server:

```sh
go run ./cmd/httpserver
```

In another terminal, send a request:

```sh
curl -i http://localhost:42069/
```

Expected behavior:

- The server listens on TCP port `42069`.
- The root route returns an HTML `200 OK` response.
- The server shuts down when it receives `SIGINT` or `SIGTERM`, for example with `Ctrl+C`.

Run the full test suite:

```sh
go test ./...
```

If your Go build cache is not writable in your environment, use a project-local cache:

```sh
mkdir -p .gocache
GOCACHE="$PWD/.gocache" go test ./...
```

## Demo Server

The main demonstration program lives in `cmd/httpserver`.

```sh
go run ./cmd/httpserver
```

It starts a custom TCP-based HTTP server on port `42069`:

```text
localhost:42069
```

### Routes

| Route | Status | Content Type | Behavior |
| --- | ---: | --- | --- |
| `/` or any unmatched path | `200 OK` | `text/html` | Returns a simple success HTML page. |
| `/yourproblem` | `400 Bad Request` | `text/html` | Returns a demo client-error HTML page. |
| `/myproblem` | `500 Internal Server Error` | `text/html` | Returns a demo server-error HTML page. |
| `/video` | `200 OK` | `video/mp4` | Reads `assets/vim.mp4` and writes it as the response body. |
| `/httpbin/*` | `200 OK` | `text/plain` default header with chunked body | Proxies the suffix to `https://httpbin.org/*`, streams the response with chunked transfer encoding, and writes trailers. |

Example requests:

```sh
curl -i http://localhost:42069/
curl -i http://localhost:42069/yourproblem
curl -i http://localhost:42069/myproblem
curl -i http://localhost:42069/video --output vim.mp4
curl -i --raw http://localhost:42069/httpbin/get
```

The `/httpbin/*` route removes `Content-Length`, sets `Transfer-Encoding: chunked`, and announces two trailers:

- `X-Content-SHA256`
- `X-Content-Length`

After streaming the proxied response, it writes the trailers with the SHA-256 hash and byte length of the streamed body.

## Command-Line Programs

### `cmd/httpserver`

The main custom HTTP server.

```sh
go run ./cmd/httpserver
```

Responsibilities:

- Opens a TCP listener on port `42069`
- Parses HTTP requests with `internal/request`
- Writes responses with `internal/response`
- Dispatches requests through a `server.Handler`
- Demonstrates fixed-length responses, static file responses, and chunked responses with trailers

### `cmd/tcplistener`

A debugging-oriented TCP listener that accepts connections on `:42069`, parses one HTTP request from each connection, and prints the parsed request line, headers, and body.

```sh
go run ./cmd/tcplistener
```

Then send a request from another terminal:

```sh
curl -i -X POST http://localhost:42069/submit \
  -H 'Content-Type: text/plain' \
  --data 'hello world'
```

This program is useful when working on the parser because it shows exactly what the custom request parser extracted.

### `cmd/udpsender`

An interactive UDP sender that writes user-entered lines to `localhost:42069`.

```sh
go run ./cmd/udpsender
```

This utility is separate from the HTTP server path. The HTTP server listens on TCP, while this program sends UDP datagrams. It is useful for socket experimentation, but it is not part of the HTTP request/response flow.

## Internal Packages

All reusable code lives under `internal`, so it is intentionally private to this module.

### `internal/headers`

The `headers` package defines:

```go
type Headers map[string]string
```

It provides:

- `NewHeaders() Headers`
- `Headers.Parse(data []byte) (n int, done bool, err error)`
- `Headers.Set(key, value string)`
- `Headers.Get(key string) (string, bool)`
- `Headers.Override(key, value string)`
- `Headers.Remove(key string)`

Header behavior:

- Header names are normalized to lowercase.
- Leading and trailing whitespace around names and values is trimmed after validation.
- Duplicate headers are joined with `, `.
- Parsing is incremental: `Parse` consumes at most one header line per call and reports how many bytes were consumed.
- A blank line, represented by `\r\n`, marks the end of the header section.
- Header field names are validated against the HTTP token character set used by RFC 9110.
- Invalid whitespace or invalid characters in a field name cause parsing to fail.

Example:

```go
h := headers.NewHeaders()
n, done, err := h.Parse([]byte("Host: localhost:42069\r\n\r\n"))
```

After this call:

- `n` is the number of bytes consumed for the parsed header line
- `done` is `false`, because only the `Host` line was consumed
- `h.Get("host")` returns `localhost:42069`

### `internal/request`

The `request` package turns bytes from an `io.Reader` into a structured request:

```go
type Request struct {
    RequestLine RequestLine
    Headers     headers.Headers
    Body        []byte
}

type RequestLine struct {
    HttpVersion   string
    RequestTarget string
    Method        string
}
```

The main entrypoint is:

```go
func RequestFromReader(reader io.Reader) (*Request, error)
```

Parser behavior:

- Reads from the provided `io.Reader` in small chunks.
- Starts with an 8-byte buffer and grows it when needed.
- Parses the request line first.
- Parses headers one line at a time.
- Parses the body only when `Content-Length` is present.
- Treats a request without `Content-Length` as having no body.
- Returns an error if EOF arrives before the parser reaches a complete request.
- Returns an error for malformed request lines, unsupported HTTP versions, malformed headers, malformed `Content-Length`, or bodies that exceed the declared `Content-Length`.

Request-line behavior:

- The request line must contain exactly three space-separated parts:

```text
METHOD request-target HTTP/1.1
```

- Methods must contain only uppercase `A` through `Z`.
- The HTTP version must be exactly `HTTP/1.1`.
- The request target is preserved as the middle field without further validation.

Supported examples:

```text
GET / HTTP/1.1
GET /coffee HTTP/1.1
POST /submit HTTP/1.1
```

Unsupported examples:

```text
/coffee HTTP/1.1
get / HTTP/1.1
GET / HTTP/1.0
GET / TCP/1.1
```

### `internal/response`

The `response` package writes HTTP responses to an `io.Writer`.

Important types and functions:

```go
type StatusCode int

const (
    StatusCodeSuccess             StatusCode = 200
    StatusCodeBadRequest          StatusCode = 400
    StatusCodeInternalServerError StatusCode = 500
)

func NewWriter(w io.Writer) *Writer
func GetDefaultHeaders(contentLen int) headers.Headers
```

`GetDefaultHeaders` returns:

- `Content-Length`
- `Connection: close`
- `Content-Type: text/plain`

`Writer` is stateful. Calls must be made in HTTP response order:

```go
w.WriteStatusLine(response.StatusCodeSuccess)
w.WriteHeaders(headers)
w.WriteBody(body)
```

For chunked responses:

```go
w.WriteStatusLine(response.StatusCodeSuccess)
w.WriteHeaders(headers)
w.WriteChunkedBody(chunk)
w.WriteChunkedBodyDone()
w.WriteTrailers(trailers)
```

State enforcement helps catch invalid response sequences. For example, writing a body before headers returns an error.

### `internal/server`

The `server` package wraps the parser and writer into a small concurrent TCP server.

```go
type Handler func(w *response.Writer, req *request.Request)

func Serve(port int, h Handler) (*Server, error)
```

Server behavior:

- Listens on the provided TCP port.
- Accepts connections in a loop.
- Handles each accepted connection in its own goroutine.
- Parses exactly one request from each connection.
- Closes the connection after handling the request.
- Sends a `400 Bad Request` response if request parsing fails.
- Stops accepting connections after `Close` is called.

Example:

```go
s, err := server.Serve(42069, func(w *response.Writer, req *request.Request) {
    body := []byte("hello\n")
    h := response.GetDefaultHeaders(len(body))

    _ = w.WriteStatusLine(response.StatusCodeSuccess)
    _ = w.WriteHeaders(h)
    _, _ = w.WriteBody(body)
})
if err != nil {
    log.Fatal(err)
}
defer s.Close()
```

## HTTP Behavior

### Request Parsing

The request parser is implemented as a small state machine:

1. `requestStateInitialized`
2. `requestStateParsingHeaders`
3. `requestStateParsingBody`
4. `requestStateDone`

The parser repeatedly reads from the stream, parses as much as possible, shifts unparsed bytes to the beginning of the buffer, and continues until the request is complete.

This design means the parser can handle request data split across arbitrary TCP read boundaries. The tests simulate this by feeding requests through readers that return only a few bytes at a time.

### Header Parsing

Header parsing consumes one CRLF-terminated line per call. The end of the header block is a standalone CRLF.

Header names are case-insensitive because all names are stored in lowercase:

```text
Host: localhost
hOsT: localhost
HOST: localhost
```

All of these are stored under `host`.

Duplicate values are combined:

```text
Set-Currency: USD
Set-Currency: BTC
```

This becomes:

```text
set-currency: USD, BTC
```

### Body Parsing

The parser reads a request body only when `Content-Length` exists.

Behavior by case:

- `Content-Length: 0` means an empty body.
- Missing `Content-Length` means no body is read.
- A shorter-than-declared body causes an incomplete request error when EOF is reached.
- A longer-than-declared body causes a `Content-Length too large` error.
- A malformed `Content-Length` value causes a parse error.

Transfer-encoded request bodies are not currently supported.

### Response Writing

The response writer serializes HTTP/1.1 responses in this order:

```text
HTTP/1.1 200 OK\r\n
header-name: header value\r\n
\r\n
response body
```

Supported built-in status codes:

- `200 OK`
- `400 Bad Request`
- `500 Internal Server Error`

The writer can also write chunked bodies:

```text
<chunk-size-hex>\r\n
<chunk-bytes>\r\n
0\r\n
<trailers>\r\n
\r\n
```

## Testing

Run all tests:

```sh
go test ./...
```

The current tests cover:

- Valid request-line parsing
- Invalid request-line shapes
- Method validation
- HTTP version validation
- Header parsing
- Empty headers
- Duplicate headers
- Case-insensitive header storage
- Malformed headers
- Missing end-of-header marker
- Request bodies with `Content-Length`
- Empty request bodies
- Missing `Content-Length`
- Incomplete bodies

The latest verification in this workspace passed with a project-local Go build cache:

```sh
GOCACHE="$PWD/.gocache" go test ./...
```

## Linting And Formatting

The repository includes `golangci.yml` with a strict linter setup. It enables the standard linter set plus additional checks for bug prevention, error handling, readability, security, and test quality.

If `golangci-lint` is installed, run:

```sh
golangci-lint run
```

Formatting tools configured in `golangci.yml`:

- `gofumpt`
- `goimports`

Standard Go formatting is still useful:

```sh
gofmt -w ./cmd ./internal
```

## Development Notes

### Port Usage

Both `cmd/httpserver` and `cmd/tcplistener` use port `42069`. Run only one of them at a time unless you change the port in the source.

### Static Asset

The `/video` route expects this file to exist relative to the process working directory:

```text
assets/vim.mp4
```

Run the server from the repository root so the file can be found:

```sh
go run ./cmd/httpserver
```

### Error Handling In Handlers

Most demo handlers call response writer methods without checking every returned error. That keeps the demo route code short, but production code should handle these errors because writes can fail when clients disconnect.

### Connection Lifetime

The server closes every connection after one parsed request and one response. It also sends `Connection: close` by default. Persistent connections and request pipelining are not implemented.

### External Proxy Dependency

The `/httpbin/*` route depends on:

```text
https://httpbin.org
```

If that service is unavailable, blocked, or slow, the route may fail or delay. Other local routes do not need network access.

## Current Limitations

This implementation deliberately supports only a small subset of HTTP/1.1:

- Inbound HTTP is parsed manually, but outbound proxying uses Go's standard `net/http` client.
- Only HTTP/1.1 request lines are accepted.
- Request methods must be uppercase alphabetic characters.
- Only `Content-Length` request bodies are supported.
- Chunked request bodies are not supported.
- Persistent connections are not supported.
- Pipelined requests are not supported.
- Header storage combines duplicates into a comma-separated string, which is correct for many headers but not every HTTP header.
- The response package has built-in reason phrases for only `200`, `400`, and `500`.
- No TLS support is provided by the custom server.
- No routing abstraction is provided beyond the demo handler's `if` statements.
- No middleware, context propagation, timeout management, or graceful in-flight request draining is implemented.
- The static video route reads the full file into memory before writing it.
- The parser does not impose maximum request-line, header, or body sizes beyond available memory.
- The demo server is intended for local experimentation, not direct exposure to the public internet.