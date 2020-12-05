package main

import (
	"context"

	"code.vegaprotocol.io/vega/cmd/vega/verify"

	"github.com/jessevdk/go-flags"
)

type VerifyCmd struct {
	Asset verify.AssetCmd `command:"asset" description:"verify an asset proposal payload"`
}

var verifyCmd VerifyCmd

func Verify(ctx context.Context, parser *flags.Parser) error {
	verifyCmd = VerifyCmd{}

	_, err := parser.AddCommand("verify", "verify vega data types", "", &verifyCmd)
	return err
}
