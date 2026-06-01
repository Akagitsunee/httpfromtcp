package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/akagitusnee/httpfromtcp/internal/headers"
	"github.com/akagitusnee/httpfromtcp/internal/request"
	"github.com/akagitusnee/httpfromtcp/internal/response"
	"github.com/akagitusnee/httpfromtcp/internal/server"
)

const port = 42069

func main() {
	server, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func handler(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin") {
		proxyHandler(w, req)
		return
	}
	if req.RequestLine.RequestTarget == "/video" {
		videoHandler(w, req)
		return
	}
	if req.RequestLine.RequestTarget == "/yourproblem" {
		handler400(w, req)
		return
	}
	if req.RequestLine.RequestTarget == "/myproblem" {
		handler500(w, req)
		return
	}
	handler200(w, req)
	return
}

func handler400(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(response.StatusCodeBadRequest)
	body := []byte(`<html>
<head>
<title>400 Bad Request</title>
</head>
<body>
<h1>Bad Request</h1>
<p>Your request honestly kinda sucked.</p>
</body>
</html>
`)
	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	w.WriteHeaders(h)
	w.WriteBody(body)
	return
}

func handler500(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(response.StatusCodeInternalServerError)
	body := []byte(`<html>
<head>
<title>500 Internal Server Error</title>
</head>
<body>
<h1>Internal Server Error</h1>
<p>Okay, you know what? This one is on me.</p>
</body>
</html>
`)
	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	w.WriteHeaders(h)
	w.WriteBody(body)
}

func handler200(w *response.Writer, _ *request.Request) {
	w.WriteStatusLine(response.StatusCodeSuccess)
	body := []byte(`<html>
<head>
<title>200 OK</title>
</head>
<body>
<h1>Success!</h1>
<p>Your request was an absolute banger.</p>
</body>
</html>
`)
	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "text/html")
	w.WriteHeaders(h)
	w.WriteBody(body)
	return
}

func proxyHandler(w *response.Writer, r *request.Request) {
	target := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin/")
	url := "https://httpbin.org/" + target
	fmt.Println("Proxying to", url)
	resp, err := http.Get(url)
	if err != nil {
		handler500(w, r)
		return
	}
	defer resp.Body.Close()

	w.WriteStatusLine(response.StatusCodeSuccess)
	h := response.GetDefaultHeaders(0)
	h.Override("Transfer-Encoding", "chunked")
	h.Remove("content-length")
	h.Override("Trailer", "X-Content-SHA256, X-Content-Length")
	w.WriteHeaders(h)

	buf := make([]byte, 1024)
	body := make([]byte, 0)
	for {
		n, err := resp.Body.Read(buf)
		fmt.Println("Read", n, "bytes")

		if n > 0 {
			_, err = w.WriteChunkedBody(buf[:n])
			if err != nil {
				fmt.Println("Error writing chunked body:", err)
				break
			}
			body = append(body, buf[:n]...)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading response body:", err)
			break
		}
	}
	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		fmt.Println("Error writing chunked body done:", err)
	}
	trailers := headers.NewHeaders()
	sha256 := fmt.Sprintf("%x", sha256.Sum256(body))
	trailers.Override("X-Content-SHA256", sha256)
	trailers.Override("X-Content-Length", fmt.Sprintf("%d", len(body)))
	err = w.WriteTrailers(trailers)
	if err != nil {
		fmt.Println("Error writing trailers:", err)
	}
}

func videoHandler(w *response.Writer, req *request.Request) {
	w.WriteStatusLine(response.StatusCodeSuccess)
	const filepath = "assets/vim.mp4"
	body, err := os.ReadFile(filepath)
	if err != nil {
		handler500(w, nil)
		return
	}

	h := response.GetDefaultHeaders(len(body))
	h.Override("Content-Type", "video/mp4")
	w.WriteHeaders(h)
	w.WriteBody(body)
	return
}
