package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"vega/proto"

	sse "github.com/alexandrevicenzi/go-sse"
)

type Server struct {
	server sse.Server
}

func NewServer(orderChan chan msg.Order, tradeChan chan msg.Trade) Server {
	s := Server{
		server: *sse.NewServer(&sse.Options{
			// CORS headers
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, OPTIONS",
				"Access-Control-Allow-Headers": "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID",
			},
		},
	)}
	go s.handleOrders(orderChan)
	go s.handleTrades(tradeChan)
	return s
}

func (s *Server) Start() {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Starting SSE server on port %d\n", port)
	defer s.server.Shutdown()

	http.Handle("/events/", &s.server)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *Server) handleOrders(orders chan msg.Order) {
	for order := range orders {
		orderJson, _ := json.Marshal(order)
		s.server.SendMessage("/events/orders", sse.SimpleMessage(string(orderJson)))
	}
}

func (s *Server) handleTrades(trades chan msg.Trade) {
	for trade := range trades {
		tradeJson, _ := json.Marshal(trade)
		s.server.SendMessage("/events/trades", sse.SimpleMessage(string(tradeJson)))
	}
}
