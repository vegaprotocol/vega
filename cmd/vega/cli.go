package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Option uses to define the global options.
type Option struct {
	Debug bool
}

type Cli struct {
	Option
	rootCmd *cobra.Command
	padding int
}

const (
	defaultVersionHash = "unknown"
	defaultVersion     = "unknown"
)

var (
	VersionHash = ""
	Version     = ""
)

var aboutVega = `
 __      __  ______    _____
 \ \    / / |  ____|  / ____|     /\
  \ \  / /  | |__    | |  __     /  \
   \ \ \/   |  __|   | | |_ |   / /\ \
    \ \     | |____  | |__| |  / ____ \
     \/     |______|  \_____| /_/    \_\

`

// NewCli creates an instance of 'Cli'.
func NewCli() *Cli {
	if len(VersionHash) <= 0 {
		VersionHash = defaultVersionHash
	}
	if len(Version) <= 0 {
		Version = defaultVersion
	}

	return &Cli{
		rootCmd: &cobra.Command{
			Use:               "vega",
			Short:             "Smart infrastructure for a better financial system.",
			Long:              aboutVega,
			DisableAutoGenTag: true,
			Version:           fmt.Sprintf("%v (%v)", Version, VersionHash),
		},
		padding: 3,
	}
}

// Run executes the client program.
func (c *Cli) Run() error {
	return c.rootCmd.Execute()
}

// AddCommand add a sub-command.
func (c *Cli) AddCommand(parent, child Command) {
	child.Init(c)

	parentCmd := parent.Cmd()
	childCmd := child.Cmd()

	// make command error not return command usage and error
	childCmd.SilenceUsage = true
	childCmd.SilenceErrors = true
	childCmd.DisableFlagsInUseLine = true

	parentCmd.AddCommand(childCmd)
}
