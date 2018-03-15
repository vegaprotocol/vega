package sse

import (
	"fmt"
	"log"
	"net/http"
	"time"

	sse "github.com/alexandrevicenzi/go-sse"
)

func NewSseServer() SseBroker {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Start SSE server on port %d", port)

	s := sse.NewServer(nil)
	defer s.Shutdown()

	sseStreamer := NewSseStreamer(*s)

	// Register with /events endpoint.
	http.Handle("/events/", s)

	// Dispatch heartbeat messages
	go func() {
		for {
			s.SendMessage("/events/heartbeat", sse.SimpleMessage(time.Now().String()))
			time.Sleep(5 * time.Second)
		}
	}()

	log.Fatal(http.ListenAndServe(addr, nil))

	return *sseStreamer
}
