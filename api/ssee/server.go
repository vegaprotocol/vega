package ssee

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
	s := sse.NewServer(nil)
	return SseServer{server: *s}
}

func (s *SseServer) Start() {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Start SSE server on port %d", port)
	defer s.server.Shutdown()

	http.Handle("/events/", &s.server)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *SseServer) SendOrder(order msg.Order) {
	var json, _ = json.Marshal(order)
	s.server.SendMessage("/events/orders", sse.SimpleMessage(string(json)))
}
