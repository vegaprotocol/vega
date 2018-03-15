package sse

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	sse "github.com/alexandrevicenzi/go-sse"
)

func NewSseServer() {
	var port = 3002
	var addr = fmt.Sprintf(":%d", port)
	fmt.Printf("Start SSE server on port %d", port)

	s := sse.NewServer(nil)
	defer s.Shutdown()

	// Register with /events endpoint.
	http.Handle("/events/", s)

	// Dispatch messages to channel-1.
	go func() {
		for {
			s.SendMessage("/events/channel-1", sse.SimpleMessage(time.Now().String()))
			time.Sleep(5 * time.Second)
		}
	}()

	// Dispatch messages to channel-2
	go func() {
		i := 0
		for {
			i++
			s.SendMessage("/events/channel-2", sse.SimpleMessage(strconv.Itoa(i)))
			time.Sleep(5 * time.Second)
		}
	}()

	http.ListenAndServe(addr, nil)
}
