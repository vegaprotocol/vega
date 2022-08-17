// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package config

import (
	"fmt"

	"code.vegaprotocol.io/vega/paths"
)

type BinaryConfig struct {
	Path string   `toml:"path"`
	Args []string `toml:"args"`
}

type RPCConfig struct {
	SocketPath string `toml:"socketPath"`
	HttpPath   string `toml:"httpPath"`
}

type VegaConfig struct {
	Binary BinaryConfig `toml:"binary"`
	RCP    RPCConfig    `toml:"rpc"`
}

type DataNodeConfig struct {
	Binary BinaryConfig `toml:"binary"`
}

type RunConfig struct {
	Name     string          `toml:"name"`
	Vega     VegaConfig      `toml:"vega"`
	DataNode *DataNodeConfig `toml:"data_node"`
}

func ExampleRunConfig(name string, withDataNode bool) *RunConfig {
	c := &RunConfig{
		Name: name,
		Vega: VegaConfig{
			Binary: BinaryConfig{
				Path: "vega",
				Args: []string{"arg1", "arg2", "..."},
			},
		},
	}

	if withDataNode {
		c.DataNode = &DataNodeConfig{
			Binary: BinaryConfig{
				Path: "data-node",
				Args: []string{"arg1", "arg2", "..."},
			},
		}
	}

	return c
}

func ParseRunConfig(path string) (*RunConfig, error) {
	conf := RunConfig{}
	if err := paths.ReadStructuredFile(path, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse RunConfig: %w", err)
	}

	return &conf, nil
}

func (rc *RunConfig) WriteToFile(path string) error {
	return paths.WriteStructuredFile(path, rc)
}
