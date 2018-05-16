package gin

import (
	"fmt"
	"net/http"
)

type restServer struct{}

type RestServer interface {
	Start()
}

func NewRestServer() *restServer {
	return &restServer{}
}

func (s *restServer) Start() {
	var port= 3001
	var addr= fmt.Sprintf(":%d", port)
	fmt.Printf("Starting based REST/JSON HTTP server on port %d...\n", port)

	router := NewRouter()
	http.ListenAndServe(addr, router)
}
