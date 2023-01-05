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

/*
description: Allows to configure binary and it's arguments.
example:

	type: toml
	value: |
		path = "/path/binary"
		args = ["--arg1", "val1", "--arg2"]
*/
type BinaryConfig struct {
	/*
		description: Path to the binary.
		note: |
			Both absolute or relative path can be used.
			Relative path is relative to a parent folder of this config file.
	*/
	Path string `toml:"path"`
	/*
		description: Arguments that will be applied to the binary.
		note: |
			Each element the list represents one space seperated argument.
	*/
	Args []string `toml:"args"`
}

/*
description: Allows to configure connection to core node exposed UNIX socket RPC API.
example:

	type: toml
	value: |
		[vega.rpc]
			socketPath = "/path/socket.sock"
			httpPath = "/rpc"
*/
type RPCConfig struct {
	/*
		description: Path of the mounted socket.
		note: This path can be configured in Vega core node configuration.
	*/
	SocketPath string `toml:"socketPath"`
	/*
		description: HTTP path of the socket path.
		note: This path can be configured in Vega core node configuration.
	*/
	HTTPPath string `toml:"httpPath"`
}

/*
description: Allows to configure Vega binary and it's arguments.
example:

	type: toml
	value: |
		[vega]
			[vega.binary]
				path = "/path/vega-binary"
				args = ["--arg1", "val1", "--arg2"]
			[vega.rpc]
				socketPath = "/path/socket.sock"
				httpPath = "/rpc"
*/
type VegaConfig struct {
	/*
		description: Configuration of Vega binary to be run.
		example:
			type: toml
			value: |
				[vega.binary]
					path = "/path/vega-binary"
					args = ["--arg1", "val1", "--arg2"]
	*/
	Binary BinaryConfig `toml:"binary"`

	/*
		description: |
			Visor communicates with the core node via RPC API that runs over UNIX socket.
			This parameter allows to configure the UNIX socket to match the core node configuration.
		example:
			type: toml
			value: |
				[vega.binary]
					path = "/path/vega-binary"
					args = ["--arg1", "val1", "--arg2"]
	*/
	RCP RPCConfig `toml:"rpc"`
}

/*
description: Allows to configure Data node binary and it's arguments.
example:

	type: toml
	value: |
		[data_node]
			[data_node.binary]
				path = "/path/data-node-binary"
				args = ["--arg1", "val1", "--arg2"]
*/
type DataNodeConfig struct {
	Binary BinaryConfig `toml:"binary"`
}

/*
description: Root of the config file
example:

	type: toml
	value: |
		name = "v1.65.0"

		[vega]
			[vega.binary]
				path = "/path/vega-binary"
				args = ["--arg1", "val1", "--arg2"]
			[vega.rpc]
				socketPath = "/path/socket.sock"
				httpPath = "/rpc"
*/
type RunConfig struct {
	/*
		description: Name of the upgrade.
		note: It is recommended to use an upgrade version as a name.
	*/
	Name string `toml:"name"`
	// description: Configuration of a Vega node.
	Vega VegaConfig `toml:"vega"`
	// description: Configuration of a Data node.
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
