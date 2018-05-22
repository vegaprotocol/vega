package rest

import (
	"fmt"
	"net/http"
	"vega/api/trading/orders"
)

type restServer struct{}

func NewRestServer() *restServer {
	return &restServer{}
}

func (s *restServer) Start() {
	var port= 3003
	var addr= fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST based HTTP server on port %d...\n", port)

	// Create dependencies
	orderService := orders.NewRpcOrderService()
	router := NewRouter(orderService)
	http.ListenAndServe(addr, router)
}