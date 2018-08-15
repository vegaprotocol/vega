package blockchain

import (
	"context"
	"vega/core"
	"vega/log"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tmlibs/common"
	"vega/tendermint/rpc"
	"time"
	"vega/msg"
	"fmt"
)

// Starts up a Vega blockchain server.
func Start(vega *core.Vega) error {
	log.Infof("Starting Vega blockchain socket...")
	blockchain := NewBlockchain(vega)
	srv, err := server.NewServer("127.0.0.1:46658", "socket", blockchain)
	if err != nil {
		fmt.Println("AAAAA")
		return err
	}
	if err := srv.Start(); err != nil {
		fmt.Println("BBBBB")
		return err
	}
	vega.Statistics.Status = msg.AppStatus_CHAIN_NOT_FOUND

	blockchainClient := NewClient()
	var genesis *rpc.Genesis
	for {
		log.Infof("Attempting to retrieve Tendermint genesis time...")
		genesis, err = blockchainClient.GetGenesisTime(context.Background())
		if genesis != nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	log.Infof("Genesis time set to: %+v\n", genesis.GenesisTime)
	vega.SetGenesisTime(genesis.GenesisTime)
	vega.Statistics.Status = msg.AppStatus_APP_CONNECTED

	// Wait forever
	cmn.TrapSignal(func() {
		srv.Stop()
	})
	return nil
}
