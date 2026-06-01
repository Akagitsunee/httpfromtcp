package main

import (
	"fmt"
	"log"
	"net"

	"github.com/akagitusnee/httpfromtcp/internal/request"
)

const port = ":42069"

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("error listening for TCP traffic: %s\n", err.Error())
	}
	defer listener.Close()

	fmt.Println("Listening for TCP traffic on", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}
		fmt.Println("Accepted connection from", conn.RemoteAddr())

		request, err := request.RequestFromReader(conn)

		fmt.Println("Request line:")
		fmt.Println("- Method:", request.RequestLine.Method)
		fmt.Println("- Target:", request.RequestLine.RequestTarget)
		fmt.Println("- Version:", request.RequestLine.HttpVersion)
		fmt.Println("Headers:")
		for key, value := range request.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}
		fmt.Println("Body:")
		fmt.Println(string(request.Body))

		fmt.Println("Connection to ", conn.RemoteAddr(), "closed")
	}
}
