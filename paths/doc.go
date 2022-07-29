package paths

import "fmt"

const (
	// LongestPathNameLen is the length of the longest path name. It is used
	// for text formatting.
	LongestPathNameLen = 35
)

type ListPathsResponse struct {
	CachePaths  map[string]string `json:"cachePaths"`
	ConfigPaths map[string]string `json:"configPaths"`
	DataPaths   map[string]string `json:"dataPaths"`
	StatePaths  map[string]string `json:"statePaths"`
}

func List(vegaPaths Paths) *ListPathsResponse {
	return &ListPathsResponse{
		CachePaths: map[string]string{
			"DataNodeCacheHome": vegaPaths.CachePathFor(DataNodeCacheHome),
		},
		ConfigPaths: map[string]string{
			"DataNodeConfigHome":                 vegaPaths.ConfigPathFor(DataNodeConfigHome),
			"DataNodeDefaultConfigFile":          vegaPaths.ConfigPathFor(DataNodeDefaultConfigFile),
			"FaucetConfigHome":                   vegaPaths.ConfigPathFor(FaucetConfigHome),
			"FaucetDefaultConfigFile":            vegaPaths.ConfigPathFor(FaucetDefaultConfigFile),
			"NodeConfigHome":                     vegaPaths.ConfigPathFor(NodeConfigHome),
			"NodeDefaultConfigFile":              vegaPaths.ConfigPathFor(NodeDefaultConfigFile),
			"NodeWalletsConfigFile":              vegaPaths.ConfigPathFor(NodeWalletsConfigFile),
			"WalletCLIConfigHome":                vegaPaths.ConfigPathFor(WalletCLIConfigHome),
			"WalletCLIDefaultConfigFile":         vegaPaths.ConfigPathFor(WalletCLIDefaultConfigFile),
			"WalletAppConfigHome":                vegaPaths.ConfigPathFor(WalletAppConfigHome),
			"WalletAppDefaultConfigFile":         vegaPaths.ConfigPathFor(WalletAppDefaultConfigFile),
			"WalletServiceConfigHome":            vegaPaths.ConfigPathFor(WalletServiceConfigHome),
			"WalletServiceNetworksConfigHome":    vegaPaths.ConfigPathFor(WalletServiceNetworksConfigHome),
			"WalletServicePermissionsConfigFile": vegaPaths.ConfigPathFor(WalletServicePermissionsConfigFile),
		},
		DataPaths: map[string]string{
			"NodeDataHome":                       vegaPaths.DataPathFor(NodeDataHome),
			"NodeWalletsDataHome":                vegaPaths.DataPathFor(NodeWalletsDataHome),
			"VegaNodeWalletsDataHome":            vegaPaths.DataPathFor(VegaNodeWalletsDataHome),
			"EthereumNodeWalletsDataHome":        vegaPaths.DataPathFor(EthereumNodeWalletsDataHome),
			"FaucetDataHome":                     vegaPaths.DataPathFor(FaucetDataHome),
			"FaucetWalletsDataHome":              vegaPaths.DataPathFor(FaucetWalletsDataHome),
			"WalletsDataHome":                    vegaPaths.DataPathFor(WalletsDataHome),
			"WalletServiceDataHome":              vegaPaths.DataPathFor(WalletServiceDataHome),
			"WalletServiceRSAKeysDataHome":       vegaPaths.DataPathFor(WalletServiceRSAKeysDataHome),
			"WalletServicePublicRSAKeyDataFile":  vegaPaths.DataPathFor(WalletServicePublicRSAKeyDataFile),
			"WalletServicePrivateRSAKeyDataFile": vegaPaths.DataPathFor(WalletServicePrivateRSAKeyDataFile),
		},
		StatePaths: map[string]string{
			"DataNodeStateHome":      vegaPaths.StatePathFor(DataNodeStateHome),
			"DataNodeLogsHome":       vegaPaths.StatePathFor(DataNodeLogsHome),
			"DataNodeStorageHome":    vegaPaths.StatePathFor(DataNodeStorageHome),
			"NodeStateHome":          vegaPaths.StatePathFor(NodeStateHome),
			"NodeLogsHome":           vegaPaths.StatePathFor(NodeLogsHome),
			"CheckpointStateHome":    vegaPaths.StatePathFor(CheckpointStateHome),
			"SnapshotStateHome":      vegaPaths.StatePathFor(SnapshotStateHome),
			"SnapshotDBStateFile":    vegaPaths.StatePathFor(SnapshotDBStateFile),
			"WalletCLIStateHome":     vegaPaths.StatePathFor(WalletCLIStateHome),
			"WalletCLILogsHome":      vegaPaths.StatePathFor(WalletCLILogsHome),
			"WalletAppStateHome":     vegaPaths.StatePathFor(WalletAppStateHome),
			"WalletAppLogsHome":      vegaPaths.StatePathFor(WalletAppLogsHome),
			"WalletServiceStateHome": vegaPaths.StatePathFor(WalletServiceStateHome),
			"WalletServiceLogsHome":  vegaPaths.StatePathFor(WalletServiceLogsHome),
		},
	}
}

func Explain(name string) (string, error) {
	paths := map[string]string{
		"DataNodeCacheHome":                  `This folder contains the cache used by the data-node.`,
		"DataNodeConfigHome":                 `This folder contains the configuration files used by the data-node.`,
		"DataNodeDefaultConfigFile":          `This file contains the configuration used by the data-node.`,
		"FaucetConfigHome":                   `This folder contains the configuration files used by the faucet.`,
		"FaucetDefaultConfigFile":            `This file contains the configuration used by the faucet.`,
		"NodeConfigHome":                     `This folder contains the configuration files used by the node.`,
		"NodeDefaultConfigFile":              `This file contains the configuration used by the node.`,
		"NodeWalletsConfigFile":              `This file contains information related to the registered node's wallets used by the node.`,
		"WalletCLIConfigHome":                `This folder contains the configuration files used by the wallet-cli.`,
		"WalletCLIDefaultConfigFile":         `This file contains the configuration used by the wallet-cli.`,
		"WalletAppConfigHome":                `This folder contains the configuration files used by the wallet-app.`,
		"WalletAppDefaultConfigFile":         `This file contains the configuration used by the wallet-app.`,
		"WalletServiceConfigHome":            `This folder contains the configuration files used by the wallet's service.`,
		"WalletServiceNetworksConfigHome":    `This folder contains the network configuration files used by the wallet's service.`,
		"WalletServicePermissionsConfigFile": `This file contains the permissions that control the access to the wallets.`,
		"NodeDataHome":                       `This folder contains the data managed by the node.`,
		"NodeWalletsDataHome":                `This folder contains the data managed by the node's wallets.`,
		"VegaNodeWalletsDataHome":            `This folder contains the Vega wallet registered as node's wallet, used by the node to sign Vega commands.`,
		"EthereumNodeWalletsDataHome":        `This folder contains the Ethereum wallet registered as node's wallet, used by the node to interact with the Ethereum blockchain.`,
		"FaucetDataHome":                     `This folder contains the data used by the faucet.`,
		"FaucetWalletsDataHome":              `This folder contains the Vega wallet used by the faucet to sign its deposit commands.`,
		"WalletsDataHome":                    `This folder contains the "user's" wallets. These wallets are used by the user to issue commands to a Vega network.`,
		"WalletServiceDataHome":              `This folder contains the data used by the wallet's service.`,
		"WalletServiceRSAKeysDataHome":       `This folder contains the RSA keys used by the wallet's service for authentication.`,
		"WalletServicePublicRSAKeyDataFile":  `This file contains the public RSA key used by the wallet's service for authentication.`,
		"WalletServicePrivateRSAKeyDataFile": `This file contains the private RSA key used by the wallet's service for authentication.`,
		"DataNodeStateHome":                  `This folder contains the state files used by the data-node.`,
		"DataNodeLogsHome":                   `This folder contains the log files generated by the data-node.`,
		"DataNodeStorageHome":                `This folder contains the consolidated state, built out of the Vega network events, and served by the data-node's API.`,
		"NodeStateHome":                      `This folder contains the state files used by the node.`,
		"NodeLogsHome":                       `This folder contains the log files generated by the node.`,
		"CheckpointStateHome":                `This folder contains the network checkpoints generated by the node.`,
		"SnapshotStateHome":                  `This folder contains the Tendermint snapshots of the application state generated by the node.`,
		"SnapshotDBStateFile":                `This file is a database containing the snapshots' data of the of the application state generated by the node`,
		"WalletCLIStateHome":                 `This folder contains the state files used by the wallet-cli.`,
		"WalletCLILogsHome":                  `This folder contains the log files generated by the wallet-cli.`,
		"WalletAppStateHome":                 `This folder contains the state files used by the wallet-app.`,
		"WalletAppLogsHome":                  `This folder contains the log files generated by the wallet-app.`,
		"WalletServiceStateHome":             `This folder contains the state files used by the wallet's service.`,
		"WalletServiceLogsHome":              `This folder contains the log files generated by the wallet's service'.`,
	}

	description, ok := paths[name]
	if !ok {
		return "", fmt.Errorf("path \"%s\" has no documentation", name)
	}

	return description, nil
}
