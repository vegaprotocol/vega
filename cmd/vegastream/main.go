package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
)

var (
	party      string
	market     string
	serverAddr string
	batchSize  int64
)

func init() {
	flag.Int64Var(&batchSize, "batch", 0, "size of the batch")
	flag.StringVar(&party, "party", "", "name of the party to listen for updates")
	flag.StringVar(&market, "market", "", "id of the market to listen for updates")
	flag.StringVar(&serverAddr, "addr", "127.0.0.1:3002", "address of the grpc server")
}

func run(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) error {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataServiceClient(conn)
	stream, err := client.ObserveEventBus(ctx)
	if err != nil {
		conn.Close()
		return err
	}

	req := &api.ObserveEventBusRequest{
		MarketID:  market,
		PartyID:   party,
		BatchSize: batchSize,
		Type:      []proto.BusEventType{proto.BusEventType_BUS_EVENT_TYPE_ALL},
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("error when sending initial message in stream: %w", err)
	}

	poll := &api.ObserveEventBusRequest{
		BatchSize: batchSize,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()
		defer stream.CloseSend()
		defer cancel()

		m := jsonpb.Marshaler{}
		for {
			o, err := stream.Recv()
			if err == io.EOF {
				log.Printf("stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("stream closed err=%v", err)
				break
			}
			for _, e := range o.Events {
				estr, err := m.MarshalToString(e)
				if err != nil {
					log.Printf("unable to marshal event err=%v", err)
				}

				fmt.Printf("%v\n", estr)
			}
			if batchSize > 0 {
				if err := stream.SendMsg(poll); err != nil {
					log.Printf("failed to poll next event batch err=%v", err)
					return
				}
			}
		}

	}()

	return nil
}

func main() {
	flag.Parse()

	if len(serverAddr) <= 0 {
		log.Printf("error: missing grpc server address")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	if err := run(ctx, cancel, &wg); err != nil {
		log.Printf("error when starting the stream: %v", err)
		os.Exit(1)
	}

	waitSig(ctx, cancel)
	wg.Wait()
}

func waitSig(ctx context.Context, cancel func()) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Printf("Caught signal name=%v", sig)
		log.Printf("closing client connections")
		cancel()
	case <-ctx.Done():
		return
	}
}
