package main

import (
	"fmt"

	"code.vegaprotocol.io/vega/version"
)

type versionCommand struct{}

func newVersion() *versionCommand {
	return &versionCommand{}
}

func (*versionCommand) Parse(_ []string) error {
	return nil
}

func (*versionCommand) Execute() error {
	fmt.Printf("Vega CLI %s (%s)\n", version.Get(), version.GetCommitHash())
	return nil
}
