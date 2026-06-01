package server

import (
	"github.com/akagitusnee/httpfromtcp/internal/request"
	"github.com/akagitusnee/httpfromtcp/internal/response"
)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Handler func(w *response.Writer, req *request.Request)
