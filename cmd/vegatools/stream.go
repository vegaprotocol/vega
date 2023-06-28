package tools

import (
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/vegatools/stream"
)

type streamCmd struct {
	config.OutputFlag
	BatchSize uint     `description:"size of the event stream batch of events" long:"batch-size"                                                                               short:"b"`
	Party     string   `description:"name of the party to listen for updates"  long:"party"                                                                                    short:"p"`
	Market    string   `description:"name of the market to listen for updates" long:"market"                                                                                   short:"m"`
	Address   string   `description:"address of the grpc server"               long:"address"                                                                                  required:"true"   short:"a"`
	LogFormat string   `default:"raw"                                          description:"output stream data in specified format. Allowed values: raw (default), text, json" long:"log-format"`
	Reconnect bool     `description:"if connection dies, attempt to reconnect" long:"reconnect"                                                                                short:"r"`
	Type      []string `default:""                                             description:"one or more event types to subscribe to (default=ALL)"                             long:"type"       short:"t"`
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
