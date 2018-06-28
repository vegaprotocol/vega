package rest

import (
	"fmt"
	"net/http"
	"vega/api/trading/orders"
	"vega/api/trading/trades"
)

type restServer struct{
	orderService orders.OrderService
	tradeService trades.TradeService
}

func NewRestServer(orderService orders.OrderService, tradeService trades.TradeService) *restServer {
	return &restServer{
		orderService: orderService,
		tradeService: tradeService,
	}
}

func (s *restServer) Start() {
	var port= 3003
	var addr= fmt.Sprintf(":%d", port)
	fmt.Printf("Starting REST based HTTP server on port %d...\n", port)
	router := NewRouter(s.orderService, s.tradeService)
	http.ListenAndServe(addr, router)
}