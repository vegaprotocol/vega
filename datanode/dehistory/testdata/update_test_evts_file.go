package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	broker2 "code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

const (
	toHeight  = 5000
	EventFile = "smoketest_to_block_5000.evts"
)

// reads in a source event file and write the events to target events file until the given block height is reached
func main() {
	if len(os.Args) != 1 {
		fmt.Printf("expected <source events file>")
	}

	sourceFile := os.Args[1]

	fmt.Printf("creating target event file to height %d")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	config := broker.NewDefaultConfig()

	fileEventSource, err := broker.NewFileEventSource(sourceFile, 0, config.FileEventSourceConfig.SendChannelBufferSize,
		"testnet")
	if err != nil {
		panic(err)
	}

	eventsCh, errCh := fileEventSource.Receive(ctx)

	fileClient, err := broker2.NewFileClient(logging.NewTestLogger(), &broker2.FileConfig{
		Enabled: true,
		File:    EventFile,
	})
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			panic(err)
		case event := <-eventsCh:
			if event.Type() == events.TimeUpdate {
				timeUpdate := event.(entities.TimeUpdateEvent)
				if timeUpdate.BlockNr() > toHeight+1 {
					fileClient.Close()
					compressEventFile(EventFile, EventFile+".gz")
					err = os.RemoveAll(EventFile)
					if err != nil {
						panic(err)
					}
					return
				}

			}

			fileClient.Send(event)
		}
	}
}

func compressEventFile(source string, target string) {
	sourceFile, err := os.Open(source)
	if err != nil {
		panic(err)
	}
	defer func() {
		sourceFile.Close()
	}()

	fileToWrite, err := os.Create(target)
	if err != nil {
		panic(err)
	}

	zw := gzip.NewWriter(fileToWrite)
	if err != nil {
		panic(err)
	}
	defer func() {
		zw.Close()
	}()

	if _, err := io.Copy(zw, sourceFile); err != nil {
		panic(err)
	}
}

type chainInfo struct {
	chainID string
}

func (t *chainInfo) SetChainID(s string) error {
	t.chainID = s
	return nil
}

func (t *chainInfo) GetChainID() (string, error) {
	return t.chainID, nil
}
