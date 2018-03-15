package rest

import (
	"fmt"
	"log"
	"net/http"
)

type restServer struct{}

func NewRestServer() *restServer {
	return &restServer{}
}

func (s *restServer) Start() {
	var port = 3001
	var addr = fmt.Sprintf(":%d", port)

	fmt.Printf("Starting REST/JSON HTTP server on port %d...", port)

	router := NewRouter()
	log.Fatal(http.ListenAndServe(addr, router))

	fmt.Println("done")
}
