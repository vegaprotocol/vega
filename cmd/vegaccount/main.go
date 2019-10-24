/*
Command vegaccount uses the gRPC call NotifyTraderAccount to add free money to trader accounts.

Syntax:

    vegaccount -traderid sometrader -addr somenode.somenet.vega.xyz:3002
*/
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
	withdraw bool
	asset    string
	amount   uint64
)

func init() {
	flag.StringVar(&addr, "addr", "localhost:3002", "address of the node grpc api")
	flag.StringVar(&traderID, "traderid", "", "traderid of the account we want to top up")
	flag.BoolVar(&withdraw, "withdraw", false, "withdraw the given amount from the trader account")
	flag.Uint64Var(&amount, "amount", 0, "amount to withdraw / topup")
	flag.StringVar(&asset, "asset", "", "asset to withdraw monies from, work in pair with withdraw")
}

func main() {
	flag.Parse()

	if len(addr) <= 0 {
		log.Printf("Error: Missing gRPC server address")
		return
	}
	if len(traderID) <= 0 {
		log.Printf("Error: Missing trader ID")
		return
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Printf("Error: Failed to contact gRPC server: %s", err.Error())
		return
	}
	defer conn.Close()

	client := api.NewTradingClient(conn)

	if !withdraw {
		req := &api.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: traderID,
				Amount:   amount,
			},
		}

		_, err = client.NotifyTraderAccount(context.Background(), req)
		if err != nil {
			log.Printf("Error: gRPC call NotifyTraderAccount failed: %s", err.Error())
			return
		}
	} else {
		if len(asset) <= 0 {
			log.Printf("Error: Missing asset with withdraw command")
			return
		}
		req := &api.WithdrawRequest{
			Withdraw: &types.Withdraw{
				PartyID: traderID,
				Asset:   asset,
				Amount:  amount,
			},
		}
		_, err = client.Withdraw(context.Background(), req)
		if err != nil {
			log.Printf("Error: gRPC call Withdraw failed: %s", err.Error())
			return
		}

	}

	log.Printf("gRPC call successfully sent for trader: %s", traderID)

}
