package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"code.vegaprotocol.io/vega/internal/api"

	"google.golang.org/grpc"
)

var (
	orders    bool
	trades    bool
	positions bool
	depth     bool
	candles   bool

	party  string
	market string

	serverAddr string

	ErrMissingMarket = errors.New("missing market parameter")
	ErrMissingParty  = errors.New("missing party parameter")
)

func init() {
	flag.BoolVar(&orders, "orders", false, "listen to newly created orders")
	flag.BoolVar(&trades, "trades", false, "listen to newly created trades")
	flag.BoolVar(&positions, "positions", false, "listen to newly created positions")
	flag.BoolVar(&depth, "depth", false, "listen to market depth")
	flag.BoolVar(&candles, "candles", false, "listen to newly created candles")
	flag.StringVar(&party, "party", "extremtrader", "name of the party to listen for updates")
	flag.StringVar(&market, "market", "BTC/DEC19", "id of the market to listen for updates")
	flag.StringVar(&serverAddr, "addr", "0.0.0.0:3003", "address of the grpc server")
}

func startOrders(ctx context.Context, wg *sync.WaitGroup) error {
	if len(market) <= 0 {
		return ErrMissingMarket
	}
	if len(party) <= 0 {
		return ErrMissinParty
	}

	conn, err := grpc.Dial(*serverAddr)
	if err != nil {
		return err
	}

	client := api.NewTradingClient(conn)
	req := &api.OrdersSubscribeRequest{
		MarketID: market,
		PartyID:  party,
	}
	stream, err := client.OrdersSubscribe(ctx, req)
	if err != nil {
		conn.Close()
		return err
	}

	go func() {
		defer wg.Done()
		defer conn.Close()
		for {
			o, err := stream.Recv()
			if err == io.EOF {
				log.Printf("orders: stream close by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("orders: stream close err=%v", err)
				break
			}
			log.Prinf("order: %v", o)
		}

	}()
	return nil
}

func startTrades(ctx context.Context, wg *sync.WaitGroup) error {
	return nil
}

func startPositions(ctx context.Context, wg *sync.WaitGroup) error {
	return nil
}

func startCandles(ctx context.Context, wg *sync.WaitGroup) error {
	return nil
}

func startDepth(ctx context.Context, wg *sync.WaitGroup) error {
	return nil
}

func main() {
	flag.Parse()

	if len(serverAddr) <= 0 {
		log.Printf("error: missing grpc server address")
		return
	}

	if !orders && !trades && !positions && !candles && !depth {
		log.Printf("error: vegastream require at least one resource to listen for")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// wait group to clean resources
	wg := sync.WaitGroup{}

	if orders {
		wg.Add(1)
		if err := startOrders(ctx, &wg); err != nil {
			log.Printf("unable to start orders err=%v", err)
			return
		}
	}

	if trades {
		wg.Add(1)
		if err := startTrades(ctx, &wg); err != nil {
			log.Printf("unable to start trades err=%v", err)
			return
		}
	}

	if positions {
		wg.Add(1)
		if err := startPositions(ctx, &wg); err != nil {
			log.Printf("unable to start positions err=%v", err)
			return
		}
	}

	if candles {
		wg.Add(1)
		if err := startCandles(ctx, &wg); err != nil {
			log.Printf("unable to start candles err=%v", err)
			return
		}
	}

	if depth {
		wg.Add(1)
		if err := startDepth(ctx, &wg); err != nil {
			log.Printf("unable to start depth err=%v", err)
			return
		}
	}

	waitSig(cancel)
	wg.Wait()
}

func waitSig(cancel func()) {
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Printf("Caught signal name=%v", sig)
		log.Printf("closing client connections")
		cancel()
	}
}
