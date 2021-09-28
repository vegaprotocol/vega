package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/bridge"

	"github.com/jessevdk/go-flags"
)

type BridgeCmd struct {
	ERC20 *bridge.ERC20Cmd `command:"erc20" description:"Validator utilities to manage the erc20 bridge"`
}

var bridgeCmd BridgeCmd

func Bridge(ctx context.Context, parser *flags.Parser) error {
	bridgeCmd = BridgeCmd{
		ERC20: bridge.ERC20(),
	}

	_, err := parser.AddCommand("bridge", "Utilities to control / manage vega bridges", "", &bridgeCmd)
	return err
}
