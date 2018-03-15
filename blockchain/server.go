package blockchain

import (
	"fmt"
	"vega/core"

	"github.com/tendermint/abci/server"
	cmn "github.com/tendermint/tmlibs/common"
)

// Starts up a Vega blockchain server.
func Start(vega core.Vega) error {
	fmt.Print("Starting Vega blockchain socket...")
	blockchain := NewBlockchain(vega)
	srv, err := server.NewServer("127.0.0.1:46658", "socket", blockchain)
	if err != nil {
		return err
	}

	if err := srv.Start(); err != nil {
		return err
	}

	fmt.Println("done")

	// Wait forever
	cmn.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
	return nil

}
