package sse

import (
	"fmt"
	"net/http"

	"vega/log"
	"vega/proto"

	sse "github.com/alexandrevicenzi/go-sse"
	"github.com/golang/protobuf/jsonpb"
)

type Server struct {
	server sse.Server
	jsonConfig jsonpb.Marshaler
}

func NewServer(orderChan <-chan msg.Order, tradeChan <-chan msg.Trade) Server {
	s := Server{
		server: *sse.NewServer(&sse.Options{
			// CORS headers
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, OPTIONS",
				"Access-Control-Allow-Headers": "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID",
			},
		}),
		jsonConfig: jsonpb.Marshaler{
		 	EmitDefaults: true,
		 	//EnumsAsInts: false,
		},
	}
	go s.handleOrders(orderChan)
	go s.handleTrades(tradeChan)
	return s
}

func (s *Server) Start() {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	log.Infof("Starting SSE server on port %d\n", port)
	defer s.server.Shutdown()

	http.Handle("/events/", &s.server)

	log.Fatalf("Failed to start SSE server: %s", http.ListenAndServe(addr, nil))
}

func (s *Server) handleOrders(orders <-chan msg.Order) {
	for order := range orders {
		orderJson, _ := s.jsonConfig.MarshalToString(&order)
		s.server.SendMessage("/events/orders", sse.SimpleMessage(string(orderJson)))
	}
}

func (s *Server) handleTrades(trades <-chan msg.Trade) {
	for trade := range trades {
		tradeJson, _ := s.jsonConfig.MarshalToString(&trade)
		s.server.SendMessage("/events/trades", sse.SimpleMessage(string(tradeJson)))
	}
}
