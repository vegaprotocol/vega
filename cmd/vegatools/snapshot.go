// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package tools

import (
	"encoding/json"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/vegatools/snapshotdb"

	tmconfig "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/store"
	"github.com/spf13/viper"
)

type snapshotCmd struct {
	config.OutputFlag
	DBPath               string `description:"path to snapshot state data"                                                  long:"db-path"           short:"d"`
	SnapshotContentsPath string `description:"path to file where to write the content of a snapshot"                        long:"snapshot-contents" short:"c"`
	BlockHeight          uint64 `description:"block-height of requested snapshot"                                           long:"block-height"      short:"b"`
	SetProtocolUpgrade   bool   `description:"set protocol-upgrade flag to true in the latest snapshot"                     long:"set-pup"           short:"p"`
	TendermintHome       string `description:"tendermint home directory, if set will print the last processed block height" long:"tendermint-home"`
}

func getLastProcessedBlock(homeDir string) (int64, error) {
	conf := tmconfig.DefaultConfig()
	if err := viper.Unmarshal(conf); err != nil {
		return 0, err
	}
	conf.SetRoot(homeDir)

	// lets get the last processed block from tendermint
	blockStoreDB, err := tmconfig.DefaultDBProvider(&tmconfig.DBContext{ID: "blockstore", Config: conf})
	if err != nil {
		return 0, err
	}
	blockStore := store.NewBlockStore(blockStoreDB)
	return blockStore.Height(), nil
}

func (opts *snapshotCmd) Execute(_ []string) error {
	if opts.SnapshotContentsPath != "" && opts.BlockHeight == 0 {
		return errors.New("must specify --block-height when using --write-payload")
	}

	db := opts.DBPath
	if opts.DBPath == "" {
		vegaPaths := paths.New(rootCmd.VegaHome)
		db = vegaPaths.StatePathFor(paths.SnapshotStateHome)
	}

	if opts.SetProtocolUpgrade {
		err := snapshotdb.SetProtocolUpgrade(paths.New(rootCmd.VegaHome))
		return err
	}

	if opts.SnapshotContentsPath != "" {
		fmt.Printf("finding payloads for block-height %d...\n", opts.BlockHeight)
		err := snapshotdb.SavePayloadsToFile(db, opts.SnapshotContentsPath, opts.BlockHeight)
		if err != nil {
			return err
		}
		fmt.Printf("payloads saved to '%s'\n", opts.SnapshotContentsPath)
		return nil
	}

	snapshots, invalid, err := snapshotdb.SnapshotData(db, opts.BlockHeight)
	if err != nil {
		return err
	}

	var lastProcessedBlock int64
	if opts.TendermintHome != "" {
		if lastProcessedBlock, err = getLastProcessedBlock(opts.TendermintHome); err != nil {
			return err
		}
	}

	if opts.Output.IsJSON() {
		o := struct {
			Snapshots          []snapshotdb.Data `json:"snapshots"`
			Invalid            []snapshotdb.Data `json:"invalidSnapshots,omitempty"`
			LastProcessedBlock int64             `json:"lastProcessedBlock,omitempty"`
		}{
			Snapshots:          snapshots,
			Invalid:            invalid,
			LastProcessedBlock: lastProcessedBlock,
		}
		b, err := json.Marshal(o)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	if lastProcessedBlock != 0 {
		fmt.Printf("\nLast processed block: %d\n", lastProcessedBlock)
	}

	fmt.Println("\nSnapshots available:", len(snapshots))
	for _, snap := range snapshots {
		fmt.Printf("\tHeight: %d, Version: %d, Size %d, Hash: %s\n", snap.Height, snap.Version, snap.Size, snap.Hash)
	}

	if len(invalid) == 0 {
		return nil
	}
	fmt.Println("Invalid snapshots:", len(invalid))
	for _, snap := range invalid {
		fmt.Printf("\tVersion: %d, Size %d, Hash: %s\n", snap.Version, snap.Size, snap.Hash)
	}

	return nil
}
