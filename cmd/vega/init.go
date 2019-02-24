package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"vega/internal"
	"vega/internal/fsutil"
	"vega/internal/logging"
	"vega/internal/storage"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type initCommand struct {
	command

	rootPath string
	force    bool
}

func (ic *initCommand) Init(c *Cli) {
	ic.cli = c
	ic.cmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a vega node",
		Long:  "Generate the mininal configuration required for a vega node to start",
		// Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ic.runInit(c)
		},
	}

	fs := ic.cmd.Flags()
	fs.StringVarP(&ic.rootPath, "root-path", "r", fsutil.DefaultRootDir(), "Path of the root directory in which the configuration will be located")
	fs.BoolVarP(&ic.force, "force", "f", false, "Erase exiting vega configuration at the specified path")

}

func (ic *initCommand) runInit(c *Cli) error {
	log := logging.NewLoggerFromEnv("dev")

	rootPathExists, err := fsutil.Exists(ic.rootPath)
	if err != nil {
		if _, ok := err.(*fsutil.NotFound); !ok {
			return err
		}
	}

	if rootPathExists && !ic.force {
		return fmt.Errorf("configuration already exists at `%v` please remove it first or re-run using -f", ic.rootPath)
	}

	if rootPathExists && ic.force {
		log.Info("removing existing configuration", zap.String("path", ic.rootPath))
		return os.RemoveAll(ic.rootPath)
	}

	// create the root
	if err := fsutil.EnsureDir(ic.rootPath); err != nil {
		return err
	}

	fullCandleStorePath := filepath.Join(ic.rootPath, storage.CandleStoreDataPath)
	fullOrderStorePath := filepath.Join(ic.rootPath, storage.OrderStoreDataPath)
	fullTradeStorePath := filepath.Join(ic.rootPath, storage.TradeStoreDataPath)

	// create sub-folders
	if err := fsutil.EnsureDir(fullCandleStorePath); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(fullOrderStorePath); err != nil {
		return err
	}
	if err := fsutil.EnsureDir(fullTradeStorePath); err != nil {
		return err
	}

	// generate a default configuration
	cfg, err := internal.DefaultConfig(log)
	if err != nil {
		return err
	}

	// write configuration to toml
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
		return err
	}

	// create the configuration file
	f, err := os.Create(filepath.Join(ic.rootPath, "config.toml"))
	if err != nil {
		return err
	}

	if _, err := f.WriteString(buf.String()); err != nil {
		return err
	}

	log.Info("configuration generated successfully", zap.String("path", ic.rootPath))

	return nil
}
