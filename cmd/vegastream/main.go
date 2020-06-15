/*
Command vegastream connects to a gRPC server and subscribes to various streams (accounts, orders, trades etc).

For the accounts subscription, specify account type, and optionally market and/or party.

For the orders and trades subscriptions, specify market and party.

For the positions subscription, specify party.

For the candles and (market) depth subscriptions, specify market.

Syntax:

    vegastream -addr somenode.somenet.vega.xyz:3002 [plus other options...]
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc"
)

var (
	orders    bool
	trades    bool
	positions bool
	depth     bool
	candles   bool
	accounts  bool
	transfers bool

	party   string
	market  string
	accType string

	serverAddr string
)

// Error for missing program arguments
var (
	ErrMissingAccountType = errors.New("missing account type")
	ErrMissingMarket      = errors.New("missing market")
	ErrMissingParty       = errors.New("missing party")
)

func init() {
	flag.BoolVar(&accounts, "accounts", false, "listen to accounts updates")
	flag.BoolVar(&transfers, "transfers", false, "listen to transfer responses updates")
	flag.BoolVar(&orders, "orders", false, "listen to newly created orders")
	flag.BoolVar(&trades, "trades", false, "listen to newly created trades")
	flag.BoolVar(&positions, "positions", false, "listen to newly created positions")
	flag.BoolVar(&depth, "depth", false, "listen to market depth")
	flag.BoolVar(&candles, "candles", false, "listen to newly created candles")

	flag.StringVar(&party, "party", "", "name of the party to listen for updates")
	flag.StringVar(&market, "market", "", "id of the market to listen for updates")
	flag.StringVar(&accType, "acctype", "NO_ACC", "type of the account we listenning for")
	flag.StringVar(&serverAddr, "addr", "127.0.0.1:3002", "address of the grpc server")
}

type TransferResponseWrapper proto.TransferResponse

func (tw TransferResponseWrapper) String() string {
	t := proto.TransferResponse(tw)
	var b strings.Builder
	fmt.Fprintf(&b, "transfers:\n")
	for _, v := range t.Transfers {
		tm := vegatime.UnixNano(v.Timestamp)
		fmt.Fprintf(&b, "from=%v to=%v amount=%v reference=%v type=%v timestamp=%v\n",
			v.FromAccount, v.ToAccount, v.Amount, v.Reference, v.Type, tm)
	}
	fmt.Fprintf(&b, "balances:\n")
	for _, v := range t.Balances {
		acc := v.Account
		fmt.Fprintf(&b, "id=%v owner=%v balance=%v asset=%v marketID=%v type=%v newbalance=%v\n",
			acc.Id, acc.Owner, acc.Balance, acc.Asset, acc.MarketID, acc.Type.String(), v.Balance)

	}
	return b.String()
}

func startTransferResponses(ctx context.Context, wg *sync.WaitGroup) error {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	stream, err := client.TransferResponsesSubscribe(ctx, &empty.Empty{})
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
				log.Printf("transfer responses: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("transfer responses: stream closed err=%v", err)
				break
			}
			log.Printf("new transfer responses: \n%v", TransferResponseWrapper(*o).String())
		}

	}()
	return nil
}

func startAccounts(ctx context.Context, wg *sync.WaitGroup) error {
	if len(accType) <= 0 {
		return ErrMissingAccountType
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	req := &api.AccountsSubscribeRequest{
		MarketID: market, // optional
		PartyID:  party,  // optional
		Type:     proto.AccountType(proto.AccountType_value[accType]),
	}
	stream, err := client.AccountsSubscribe(ctx, req)
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
				log.Printf("accounts: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("accounts: stream closed err=%v", err)
				break
			}
			log.Printf("account: %v", o)
		}

	}()
	return nil
}

type OrdersWrapper api.OrdersStream

func (ow OrdersWrapper) String() string {
	o := api.OrdersStream(ow)
	var b strings.Builder
	for _, v := range o.Orders {
		ts := vegatime.UnixNano(v.CreatedAt)
		tsat := vegatime.UnixNano(v.ExpiresAt)
		fmt.Fprintf(&b, "id=%v marketID=%v partyID=%v price=%v size=%v remaining=%v TIF=%v type=%v createdAt=%v status=%v expiresAt=%v reference=%v\n",
			v.Id, v.MarketID, v.PartyID, v.Price, v.Size, v.Remaining, v.TimeInForce.String(),
			v.Type.String(), ts, v.Status.String(), tsat, v.Reference,
		)

	}
	return b.String()
}

func startOrders(ctx context.Context, wg *sync.WaitGroup) error {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
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
				log.Printf("orders: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("orders: stream closed err=%v", err)
				break
			}
			log.Printf("new orders: \n%v", OrdersWrapper(*o).String())
		}

	}()
	return nil
}

type TradesWrapper api.TradesStream

func (tw TradesWrapper) String() string {
	t := api.TradesStream(tw)
	var b strings.Builder
	for _, v := range t.Trades {
		ts := vegatime.UnixNano(v.Timestamp)
		fmt.Fprintf(&b, "id=%v marketID=%v price=%v size=%v buyer=%v seller=%v buyOrder=%v sellOrder=%v timestamp=%v\n",
			v.Id, v.MarketID, v.Price, v.Size, v.Buyer, v.Seller,
			v.BuyOrder, v.SellOrder, ts,
		)

	}
	return b.String()
}

func startTrades(ctx context.Context, wg *sync.WaitGroup) error {

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	req := &api.TradesSubscribeRequest{
		MarketID: market,
		PartyID:  party,
	}
	stream, err := client.TradesSubscribe(ctx, req)
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
				log.Printf("trades: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("trades: stream closed err=%v", err)
				break
			}
			log.Printf("new trades: \n%v", TradesWrapper(*o).String())
		}

	}()
	return nil
}

func startPositions(ctx context.Context, wg *sync.WaitGroup) error {
	if len(party) <= 0 {
		return ErrMissingParty
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	req := &api.PositionsSubscribeRequest{
		PartyID: party,
	}
	stream, err := client.PositionsSubscribe(ctx, req)
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
				log.Printf("positions: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("positions: stream closed err=%v", err)
				break
			}
			log.Printf("position: %v", o)
		}

	}()
	return nil

}

func startCandles(ctx context.Context, wg *sync.WaitGroup) error {
	if len(market) <= 0 {
		return ErrMissingMarket
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	req := &api.CandlesSubscribeRequest{
		MarketID: market,
		Interval: proto.Interval_INTERVAL_I1M,
	}
	stream, err := client.CandlesSubscribe(ctx, req)
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
				log.Printf("candles: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("candles: stream closed err=%v", err)
				break
			}
			log.Printf("candles: %v", o)
		}

	}()
	return nil
}

func startDepth(ctx context.Context, wg *sync.WaitGroup) error {
	if len(market) <= 0 {
		return ErrMissingMarket
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := api.NewTradingDataClient(conn)
	req := &api.MarketDepthSubscribeRequest{
		MarketID: market,
	}
	stream, err := client.MarketDepthSubscribe(ctx, req)
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
				log.Printf("depth: stream closed by server err=%v", err)
				break
			}
			if err != nil {
				log.Printf("depth: stream closed err=%v", err)
				break
			}
			log.Printf("depth: %v", o)
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

	if !orders && !trades && !positions && !candles && !depth && !accounts && !transfers {
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

	if accounts {
		wg.Add(1)
		if err := startAccounts(ctx, &wg); err != nil {
			log.Printf("unable to start accounts err=%v", err)
			return
		}
	}

	if transfers {
		wg.Add(1)
		if err := startTransferResponses(ctx, &wg); err != nil {
			log.Printf("unable to start transfer responses err=%v", err)
			return
		}
	}

	waitSig(cancel)
	wg.Wait()
}

func waitSig(cancel func()) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	sig := <-gracefulStop
	log.Printf("Caught signal name=%v", sig)
	log.Printf("closing client connections")
	cancel()
}
