package blockchain

import (
	"context"
	"time"
	"vega/core"
	"vega/log"

	"github.com/tendermint/tendermint/abci/server"
	cmn "github.com/tendermint/tmlibs/common"
	"vega/tendermint/rpc"
)

// Starts up a Vega blockchain server.
func Start(vega *core.Vega) error {
	log.Infof("Starting Vega blockchain socket...")
	blockchain := NewBlockchain(vega)
	srv, err := server.NewServer("127.0.0.1:46658", "socket", blockchain)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	blockchainClient := NewClient()
	var genesis *rpc.Genesis
	for {
		time.Sleep(1 * time.Second)
		genesis, err = blockchainClient.GetGenesisTime(context.Background())
		if genesis != nil {
			break
		}
	}
	log.Infof("Genesis time set to: %+v\n", genesis.GenesisTime)
	vega.SetGenesisTime(genesis.GenesisTime)

	// Wait forever
	cmn.TrapSignal(func() {
		srv.Stop()
	})
	return nil
}
