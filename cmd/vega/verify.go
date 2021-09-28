package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/verify"

	"github.com/jessevdk/go-flags"
)

type VerifyCmd struct {
	Asset   verify.AssetCmd   `command:"passet" description:"verify an asset proposal payload"`
	Genesis verify.GenesisCmd `command:"genesis" description:"verify a vega genesis app state"`
}

var verifyCmd VerifyCmd

func Verify(ctx context.Context, parser *flags.Parser) error {
	verifyCmd = VerifyCmd{}

	_, err := parser.AddCommand("verify", "verify vega commands and genesis state", "", &verifyCmd)
	return err
}
