package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/cmd/examples/nullchain/faucet"
	timefoward "code.vegaprotocol.io/vega/cmd/examples/nullchain/timeforward"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"

	datanode "code.vegaprotocol.io/protos/data-node/api/v1"
)

// Mint some funds for some parties
func fillAccounts(w *Wallets, c *Connection) {
	for _, party := range w.GetParties() {
		faucet.Mint(party.pubkey, "10000000000", config.GoveranceAsset)
		faucet.Mint(party.pubkey, "10000000000", config.NormalAsset)
		timefoward.MoveByDuration(config.BlockDuration)
	}
	timefoward.MoveByDuration(config.BlockDuration)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var err error
	defer func() {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	conn, err := NewConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	vt, err := conn.VegaTime()
	if err != nil {
		return
	}
	fmt.Printf("Starting Vegatime: %s\n", vt)

	w := NewWallet(config.WalletFolder, config.Passphrase)

	// Use the faucet to deposit some funds
	fillAccounts(w, conn)

	proposeMarket := true
	doTrading := true
	settleMarket := true
	if proposeMarket {
		proposeAndVoteMarket(w, conn)
	}
	if doTrading {
		doSomeTrading(w, conn)
	}
	if settleMarket {

		parties := w.GetParties()
		term := OracleTxn("trading.termination", "true")
		err = w.SubmitTransaction(conn, parties[0], term)
		settle := OrcaleTxn(strings.Join([]string{"prices", config.NormalAsset, "value"}, "."), "1000")
		err = w.SubmitTransaction(conn, parties[0], settle)
		timefoward.MoveByDuration(5 * config.BlockDuration)

		_, err := conn.datanode.Markets(context.Background(), &datanode.MarketsRequest{})
		if err != nil {
			return
		}
	}

	vt, err = conn.VegaTime()
	if err != nil {
		return
	}
	fmt.Printf("Finished Vegatime: %s\n", vt)
}
