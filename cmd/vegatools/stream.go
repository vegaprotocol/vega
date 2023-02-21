package tools

import (
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/vegatools/stream"
)

type streamCmd struct {
	config.OutputFlag
	BatchSize uint     `short:"b" long:"batch-size" description:"size of the event stream batch of events"`
	Party     string   `short:"p" long:"party" description:"name of the party to listen for updates"`
	Market    string   `short:"m" long:"market" description:"name of the market to listen for updates"`
	Address   string   `short:"a" long:"address" required:"true" description:"address of the grpc server"`
	LogFormat string   `long:"log-format" default:"raw" description:"output stream data in specified format. Allowed values: raw (default), text, json"`
	Reconnect bool     `short:"r" long:"reconnect" description:"if connection dies, attempt to reconnect"`
	Type      []string `short:"t" long:"type" default:"" description:"one or more event types to subscribe to (default=ALL)"`
}

func (opts *streamCmd) Execute(_ []string) error {
	return stream.Run(
		opts.BatchSize,
		opts.Party,
		opts.Market,
		opts.Address,
		opts.LogFormat,
		opts.Reconnect,
		opts.Type,
	)
}
