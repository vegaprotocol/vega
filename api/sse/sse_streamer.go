package sse

import (
	"time"

	sse "github.com/alexandrevicenzi/go-sse"
)

type SseBroker struct {
	server sse.Server
}

func NewSseStreamer(s sse.Server) *SseBroker {
	return &SseBroker{server: s}
}

func (streamer *SseBroker) SendTrade(trade string) {
	streamer.server.SendMessage("/events/heartbeat", sse.SimpleMessage(time.Now().String()))
}
