package gin

import (
	"fmt"
	"net/http"
	"vega/api/services"
)

type restServer struct{}

func NewRestServer() *restServer {
	return &restServer{}
}

func (s *restServer) Start() {

	var port= 3003
	var addr= fmt.Sprintf(":%d", port)
	fmt.Printf("Starting based REST/JSON HTTP server on port %d...\n", port)

	// Create dependencies
	orderService := services.NewRpcOrderService()
	router := NewRouter(&orderService)
	http.ListenAndServe(addr, router)
}
