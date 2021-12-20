package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/verify"

	"github.com/jessevdk/go-flags"
)

type VerifyCmd struct {
	Asset   verify.AssetCmd   `command:"passet" description:"verify the payload of an asset proposal"`
	Genesis verify.GenesisCmd `command:"genesis" description:"verify the appstate of a genesis file"`
}

var verifyCmd VerifyCmd

func Verify(ctx context.Context, parser *flags.Parser) error {
	verifyCmd = VerifyCmd{}

	_, err := parser.AddCommand("verify", "Verify Vega payloads or genesis appstate", "", &verifyCmd)
	return err
}
