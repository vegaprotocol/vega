package main

import (
	"context"
	"flag"
	"log"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"

	"google.golang.org/grpc"
)

var (
	addr     string
	traderID string
)

func init() {
	flag.StringVar(&addr, "addr", "0.0.0.0:3002", "address of the node grpc api")
	flag.StringVar(&traderID, "traderid", "", "traderid of the account we want to top up")
}

func main() {
	flag.Parse()

	if len(addr) <= 0 {
		log.Printf("error: missing grpc server address")
		return
	}
	if len(traderID) <= 0 {
		log.Printf("error: missing traderID")
		return
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Printf("error: unable to dial with grpc server (err=%v)", err)
		return
	}
	defer conn.Close()

	client := api.NewTradingClient(conn)
	req := &api.NotifyTraderAccountRequest{
		Notif: &types.NotifyTraderAccount{
			TraderID: traderID,
		},
	}

	_, err = client.NotifyTraderAccount(context.Background(), req)
	if err != nil {
		log.Printf("error: grpc shite (err=%v)", err)
		return
	}
	log.Printf("trader account request sent with success")
}
