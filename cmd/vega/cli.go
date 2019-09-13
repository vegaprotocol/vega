package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Option defines the global options.
type Option struct {
	Debug bool
}

// Cli defines the client.
type Cli struct {
	Option
	rootCmd     *cobra.Command
	padding     int
	versionHash string
	version     string
}

const (
	defaultVersionHash = "unknown"
	defaultVersion     = "unknown"
)

var (
	// VersionHash specifies the git commit used to build the application. See VERSION_HASH in Makefile for details.
	VersionHash = ""

	// Version specifies the version used to build the application. See VERSION in Makefile for details.
	Version = ""
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

	// Prefix all VEGA environment variables to prevent collisions
	viper.SetEnvPrefix("VEGA")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	return &Cli{
		rootCmd: &cobra.Command{
			Use:               "vega",
			Short:             "Smart infrastructure for a better financial system.",
			Long:              aboutVega,
			DisableAutoGenTag: true,
			Version:           fmt.Sprintf("%v (%v)", Version, VersionHash),
		},
		padding:     3,
		version:     Version,
		versionHash: VersionHash,
	}
}

// Run executes the client program.
func (c *Cli) Run() error {
	return c.rootCmd.Execute()
}

// AddCommand adds a sub-command.
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
