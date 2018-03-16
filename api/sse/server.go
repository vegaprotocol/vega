package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"vega/proto"

	sse "github.com/alexandrevicenzi/go-sse"
)

type SseServer struct {
	server sse.Server
}

func NewSseServer() SseServer {
	s := sse.NewServer(&sse.Options{
		// CORS headers
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Allow-Headers": "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID",
		},
	})
	return SseServer{server: *s}
}

func (s *SseServer) Start() {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Starting SSE server on port %d\n", port)
	defer s.server.Shutdown()

	http.Handle("/events/", &s.server)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *SseServer) SendOrder(order msg.Order) {
	var json, _ = json.Marshal(order)
	s.server.SendMessage("/events/orders", sse.SimpleMessage(string(json)))
}

func (s *SseServer) SendTrade(trade msg.Trade) {
	var json, _ = json.Marshal(trade)
	s.server.SendMessage("/events/trades", sse.SimpleMessage(string(json)))
}
