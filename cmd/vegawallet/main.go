package main

import (
	"os"

	cmd "code.vegaprotocol.io/vega/cmd/vegawallet/commands"
)

func main() {
	writer := &cmd.Writer{
		Out: os.Stdout,
		Err: os.Stderr,
	}
	cmd.Execute(writer)
}
