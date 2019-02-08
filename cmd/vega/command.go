package main

import (
	"github.com/spf13/cobra"
)

type Command interface {
	Init(*Cli)
	Cmd() *cobra.Command
}

type command struct {
	cmd *cobra.Command
	cli *Cli
}

func (b *command) Init(cli *Cli) {}

func (b *command) Cmd() *cobra.Command {
	return b.cmd
}
