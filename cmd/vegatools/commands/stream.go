package cmd

import (
	"github.com/spf13/cobra"

	"code.vegaprotocol.io/vega/vegatools/stream"
)

var (
	streamOpts struct {
		batchSize  uint
		party      string
		market     string
		serverAddr string
		logFormat  string
		reconnect  bool
		types      []string
	}

	streamCmd = &cobra.Command{
		Use:   "stream",
		Short: "Stream events from vega node",
		RunE:  runStream,
	}
)

func init() {
	streamCmd.Flags().UintVarP(&streamOpts.batchSize, "batch-size", "b", 0, "size of the event stream batch of events")
	streamCmd.Flags().StringVarP(&streamOpts.party, "party", "p", "", "name of the party to listen for updates")
	streamCmd.Flags().StringVarP(&streamOpts.market, "market", "m", "", "name of the market to listen for updates")
	streamCmd.Flags().StringVarP(&streamOpts.serverAddr, "address", "a", "", "address of the grpc server")
	streamCmd.Flags().StringVar(&streamOpts.logFormat, "log-format", "raw", "output stream data in specified format. Allowed values: raw (default), text, json")
	streamCmd.Flags().BoolVarP(&streamOpts.reconnect, "reconnect", "r", false, "if connection dies, attempt to reconnect")
	streamCmd.Flags().StringSliceVarP(&streamOpts.types, "type", "t", nil, "one or more event types to subscribe to (default=ALL)")
	_ = streamCmd.MarkFlagRequired("address")
}

func runStream(*cobra.Command, []string) error {
	return stream.Run(streamOpts.batchSize,
		streamOpts.party,
		streamOpts.market,
		streamOpts.serverAddr,
		streamOpts.logFormat,
		streamOpts.reconnect,
		streamOpts.types,
	)
}
