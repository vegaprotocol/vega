package rest

import (
	"fmt"
	"net/http"
	"vega/api/trading/orders"
)

type restServer struct{
	orderService orders.OrderService
}

func NewRestServer(orderService orders.OrderService) *restServer {
	return &restServer{
		orderService: orderService,
	}
}

func (s *restServer) Start() {
	var port= 3003
	var addr= fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST based HTTP server on port %d...\n", port)
	router := NewRouter(s.orderService)
	http.ListenAndServe(addr, router)
}