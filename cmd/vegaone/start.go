package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/cmd/vegaone/config"
	"code.vegaprotocol.io/vega/cmd/vegaone/start"
	"code.vegaprotocol.io/vega/paths"
)

type startFlags struct {
	globalFlags

	Passphrase string
	NetworkURL string
	Network    string
}

func (g *startFlags) Register(fset *flag.FlagSet) {
	g.globalFlags.Register(fset)
	fset.StringVar(&g.Passphrase, "nodewallet-passphrase-file", "", "an optional file containing the passphrase for the node wallet")
	fset.StringVar(&g.Network, "network", "", "a vega network name to be started")
	fset.StringVar(&g.NetworkURL, "network-url", "", "the url from which to retrieve the genesis file of a network")

}

type startCommand struct {
	flags startFlags
	fset  *flag.FlagSet
}

func newStart() (i *startCommand) {
	defer func() { i.flags.Register(i.fset) }()
	return &startCommand{
		flags: startFlags{},
		fset:  flag.NewFlagSet("start", flag.ExitOnError),
	}
}

func (s *startCommand) Parse(args []string) error {
	return s.fset.Parse(args)
}

func (s *startCommand) Execute() error {
	if len(s.flags.Network) > 0 && len(s.flags.NetworkURL) > 0 {
		return errors.New("cannot set both -network and -network-url flags")
	}

	home := os.ExpandEnv(s.flags.Home)
	tendermintHome := filepath.Join(home, "tendermint")
	vegaPaths := paths.New(home)

	c, err := config.Load(home)
	if err != nil {
		return fmt.Errorf("couldn't load vegaone configuration: %w", err)
	}

	return start.Run(vegaPaths, tendermintHome, s.flags.NetworkURL, s.flags.Network, s.flags.Passphrase, c)
}
