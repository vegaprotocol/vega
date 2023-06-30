package paths

import "path/filepath"

// VegaHome is the name of the Vega folder for every type of file structure.
var VegaHome = "vega"

// File structure for cache
//
// CACHE_PATH
// 	└── data-node/

type CachePath string

func (p CachePath) String() string {
	return string(p)
}

// JoinCachePath joins any number of path elements with a root CachePath into a
// single path, separating them with an OS specific Separator, and returns it
// as a CachePath.
func JoinCachePath(p CachePath, elem ...string) CachePath {
	return CachePath(JoinCachePathStr(p, elem...))
}

// JoinCachePathStr joins any number of path elements with a root CachePath
// into a single path, separating them with an OS specific Separator, and
// returns it as a string.
func JoinCachePathStr(p CachePath, elem ...string) string {
	return filepath.Join(append([]string{string(p)}, elem...)...)
}

// DataNodeCacheHome is the folder containing the cache used by the
// data-node.
var DataNodeCacheHome = CachePath("data-node")

// File structure for configuration
//
// CONFIG_PATH
// 	├── data-node/
// 	│	└── config.toml
// 	├── faucet/
// 	│	└── config.toml
// 	├── node/
// 	│	├── config.toml
// 	│	└── wallets.toml
// 	├── wallet-cli/
// 	│	└── config.toml
// 	├── wallet-app/
// 	│	├── config.fairground.toml
// 	│	└── config.toml
// 	└── wallet-service/
//		├── config.toml
// 		└──  networks/

type ConfigPath string

func (p ConfigPath) String() string {
	return string(p)
}

// JoinConfigPath joins any number of path elements with a root ConfigPath into a
// single path, separating them with an OS specific Separator, and returns it
// as a ConfigPath.
func JoinConfigPath(p ConfigPath, elem ...string) ConfigPath {
	return ConfigPath(JoinConfigPathStr(p, elem...))
}

// JoinConfigPathStr joins any number of path elements with a root ConfigPath
// into a single path, separating them with an OS specific Separator, and
// returns it as a string.
func JoinConfigPathStr(p ConfigPath, elem ...string) string {
	return filepath.Join(append([]string{string(p)}, elem...)...)
}

var (
	// BlockExplorerConfigHome is the folder containing the configuration files
	// used by the block explorer.
	BlockExplorerConfigHome = ConfigPath("blockexplorer")

	// BlockExplorerDefaultConfigFile is the default configuration file for the
	// block explorer.
	BlockExplorerDefaultConfigFile = JoinConfigPath(BlockExplorerConfigHome, "config.toml")

	// DataNodeConfigHome is the folder containing the configuration files
	// used by the node.
	DataNodeConfigHome = ConfigPath("data-node")

	// DataNodeDefaultConfigFile is the default configuration file for the
	// data-node.
	DataNodeDefaultConfigFile = JoinConfigPath(DataNodeConfigHome, "config.toml")

	// FaucetConfigHome is the folder containing the configuration files
	// used by the node.
	FaucetConfigHome = ConfigPath("faucet")

	// FaucetDefaultConfigFile is the default configuration file for the
	// data-node.
	FaucetDefaultConfigFile = JoinConfigPath(FaucetConfigHome, "config.toml")

	// NodeConfigHome is the folder containing the configuration files used by
	// the node.
	NodeConfigHome = ConfigPath("node")

	// NodeDefaultConfigFile is the default configuration file for the node.
	NodeDefaultConfigFile = JoinConfigPath(NodeConfigHome, "config.toml")

	// NodeWalletsConfigFile is the configuration file for the node wallets.
	NodeWalletsConfigFile = JoinConfigPath(NodeConfigHome, "wallets.encrypted")

	// WalletCLIConfigHome is the folder containing the configuration files
	// used by the wallet CLI.
	WalletCLIConfigHome = ConfigPath("wallet-cli")

	// WalletCLIDefaultConfigFile is the default configuration file for the
	// wallet CLI.
	WalletCLIDefaultConfigFile = JoinConfigPath(WalletCLIConfigHome, "config.toml")

	// WalletAppConfigHome is the folder containing the configuration files
	// used by the wallet app application.
	WalletAppConfigHome = ConfigPath("wallet-app")

	// WalletAppFairgroundConfigFile is the Fairground configuration file for the
	// wallet app application.
	WalletAppFairgroundConfigFile = JoinConfigPath(WalletAppConfigHome, "config.fairground.toml")

	// WalletAppDefaultConfigFile is the default configuration file for the
	// wallet app application.
	WalletAppDefaultConfigFile = JoinConfigPath(WalletAppConfigHome, "config.toml")

	// WalletServiceConfigHome is the folder containing the configuration files
	// used by the wallet service.
	WalletServiceConfigHome = ConfigPath("wallet-service")

	// WalletServiceDefaultConfigFile is the default configuration file for the
	// wallet service.
	WalletServiceDefaultConfigFile = JoinConfigPath(WalletServiceConfigHome, "config.toml")

	// WalletServiceNetworksConfigHome is the folder containing the network
	// configuration files used to connect to a network.
	WalletServiceNetworksConfigHome = JoinConfigPath(WalletServiceConfigHome, "networks")
)

// File structure for data
//
// DATA_PATH
// 	├── node/
// 	│	└── wallets/
// 	│		├── vega/
// 	│		│	└── vega.<timestamp>
// 	│		└── ethereum/
// 	│			└── eth-node-wallet
// 	├── faucet/
// 	│	└── wallets/
// 	│		└── vega.<timestamp>
// 	├── wallets/
// 	│	├── vega-wallet-1
// 	│	└── vega-wallet-2
// 	└── wallet-service/
//      ├── tokens.json
//      ├── sessions.toml
// 		└── rsa-keys/
// 			├── private.pem
// 			└── public.pem

type DataPath string

func (p DataPath) String() string {
	return string(p)
}

// JoinDataPath joins any number of path elements with a root DataPath into a
// single path, separating them with an OS specific Separator, and returns it
// as a DataPath.
func JoinDataPath(p DataPath, elem ...string) DataPath {
	return DataPath(JoinDataPathStr(p, elem...))
}

// JoinDataPathStr joins any number of path elements with a root DataPath
// into a single path, separating them with an OS specific Separator, and
// returns it as a string.
func JoinDataPathStr(p DataPath, elem ...string) string {
	return filepath.Join(append([]string{string(p)}, elem...)...)
}

var (
	// NodeDataHome is the folder containing the data used by the node.
	NodeDataHome = DataPath("node")

	// NodeWalletsDataHome is the folder containing the data used by the
	// node wallets.
	NodeWalletsDataHome = DataPath(filepath.Join(NodeDataHome.String(), "wallets"))

	// VegaNodeWalletsDataHome is the folder containing the vega wallet
	// used by the node.
	VegaNodeWalletsDataHome = DataPath(filepath.Join(NodeWalletsDataHome.String(), "vega"))

	// EthereumNodeWalletsDataHome is the folder containing the ethereum wallet
	// used by the node.
	EthereumNodeWalletsDataHome = DataPath(filepath.Join(NodeWalletsDataHome.String(), "ethereum"))

	// FaucetDataHome is the folder containing the data used by the faucet.
	FaucetDataHome = DataPath("faucet")

	// FaucetWalletsDataHome is the folder containing the data used by the
	// faucet wallets.
	FaucetWalletsDataHome = DataPath(filepath.Join(FaucetDataHome.String(), "wallets"))

	// WalletsDataHome is the folder containing the user wallets.
	WalletsDataHome = DataPath("wallets")

	// WalletServiceDataHome is the folder containing the data used by the
	// wallet service.
	WalletServiceDataHome = DataPath("wallet-service")

	// WalletServiceAPITokensDataFile is the file containing all the API tokens
	// used by the third-party applications to connect to the wallet API.
	WalletServiceAPITokensDataFile = DataPath(filepath.Join(WalletServiceDataHome.String(), "tokens.json"))

	// WalletServiceSessionTokensDataFile is the file containing all the session tokens
	// generated to initiates the connection with the third-party applications to
	// connect to the wallet API.
	WalletServiceSessionTokensDataFile = DataPath(filepath.Join(WalletServiceDataHome.String(), "sessions.toml"))

	// WalletServiceRSAKeysDataHome is the folder containing the RSA keys used by
	// the wallet service.
	WalletServiceRSAKeysDataHome = DataPath(filepath.Join(WalletServiceDataHome.String(), "rsa-keys"))

	// WalletServicePublicRSAKeyDataFile is the file containing the public RSA key
	// used by the wallet service.
	WalletServicePublicRSAKeyDataFile = DataPath(filepath.Join(WalletServiceRSAKeysDataHome.String(), "public.pem"))

	// WalletServicePrivateRSAKeyDataFile is the file containing the private RSA key
	// used by the wallet service.
	WalletServicePrivateRSAKeyDataFile = DataPath(filepath.Join(WalletServiceRSAKeysDataHome.String(), "private.pem"))
)

// File structure for state
//
// STATE_HOME
// 	├── data-node/
// 	│	├── archivedeventbuffers/
// 	│	├── autocert/
// 	│	├── eventsbuffer/
// 	│	├── logs/
// 	│	├── networkhistory/
// 	│	│	├── snapshotscopyfrom/
// 	│	│	└── snapshotscopyto/
// 	│	└── storage/
// 	│		├── postgres/
// 	│		└── sqlstore/
// 	│			└── node-data/
// 	├── node/
// 	│	├── logs/
// 	│	├── checkpoints/
// 	│	└── snapshots/
// 	│		└── ldb
// 	├── wallet-cli/
// 	│	└── logs/
// 	├── wallet-app/
// 	│	└── logs/
// 	└── wallet-service/
// 		└── logs/

type StatePath string

func (p StatePath) String() string {
	return string(p)
}

// JoinStatePath joins any number of path elements with a root StatePath into a
// single path, separating them with an OS specific Separator, and returns it
// as a StatePath.
func JoinStatePath(p StatePath, elem ...string) StatePath {
	return StatePath(JoinStatePathStr(p, elem...))
}

// JoinStatePathStr joins any number of path elements with a root StatePath
// into a single path, separating them with an OS specific Separator, and
// returns it as a string.
func JoinStatePathStr(p StatePath, elem ...string) string {
	return filepath.Join(append([]string{string(p)}, elem...)...)
}

var (
	// DataNodeStateHome is the folder containing the state used by the
	// data-node.
	DataNodeStateHome = StatePath("data-node")

	// DataNodeAutoCertHome is the folder containing the automatically generated SSL certificates.
	DataNodeAutoCertHome = StatePath(filepath.Join(DataNodeStateHome.String(), "autocert"))

	// DataNodeLogsHome is the folder containing the logs of the data-node.
	DataNodeLogsHome = StatePath(filepath.Join(DataNodeStateHome.String(), "logs"))

	// DataNodeStorageHome is the folder containing the data storage of the
	// data-node.
	DataNodeStorageHome = StatePath(filepath.Join(DataNodeStateHome.String(), "storage"))

	// DataNodeStorageSQLStoreHome is the folder containing the data of the
	// SQL store.
	DataNodeStorageSQLStoreHome = StatePath(filepath.Join(DataNodeStateHome.String(), "sqlstore"))

	// DataNodeStorageSQLStoreNodeDataHome is the folder containing the data of the
	// SQL store.
	DataNodeStorageSQLStoreNodeDataHome = StatePath(filepath.Join(DataNodeStorageSQLStoreHome.String(), "node-data"))

	// DataNodeEmbeddedPostgresRuntimeDir is the runtime directory for embedded postgres.
	DataNodeEmbeddedPostgresRuntimeDir = StatePath(filepath.Join(DataNodeStorageHome.String(), "postgres"))

	// DataNodeNetworkHistoryHome is the folder containing the network history data.
	DataNodeNetworkHistoryHome = StatePath(filepath.Join(DataNodeStateHome.String(), "networkhistory"))

	// DataNodeNetworkHistorySnapshotCopyTo is the folder in which the datanode creates snapshots.
	DataNodeNetworkHistorySnapshotCopyTo = StatePath(filepath.Join(DataNodeNetworkHistoryHome.String(), "snapshotscopyto"))

	// DataNodeNetworkHistorySnapshotCopyFrom is the folder from which the datanode reads snapshot data.
	DataNodeNetworkHistorySnapshotCopyFrom = StatePath(filepath.Join(DataNodeNetworkHistoryHome.String(), "snapshotscopyfrom"))

	// DataNodeEventBufferHome is the folder containing event buffer files.
	DataNodeEventBufferHome = StatePath(filepath.Join(DataNodeStateHome.String(), "eventsbuffer"))

	// DataNodeArchivedEventBufferHome is the folder containing archived event buffer files.
	DataNodeArchivedEventBufferHome = StatePath(filepath.Join(DataNodeStateHome.String(), "archivedeventbuffers"))

	// NodeStateHome is the folder containing the state of the node.
	NodeStateHome = StatePath("node")

	// NodeLogsHome is the folder containing the logs of the node.
	NodeLogsHome = StatePath(filepath.Join(NodeStateHome.String(), "logs"))

	// CheckpointStateHome is the folder containing the checkpoint files
	// of to the node.
	CheckpointStateHome = StatePath(filepath.Join(NodeStateHome.String(), "checkpoints"))

	// SnapshotStateHome is the folder containing the snapshot files
	// of to the node.
	SnapshotStateHome = StatePath(filepath.Join(NodeStateHome.String(), "snapshots"))

	// SnapshotDBStateFile is the DB file for GoLevelDB used in snapshots.
	SnapshotDBStateFile = StatePath(filepath.Join(SnapshotStateHome.String(), "snapshot.db"))

	// SnapshotMetadataDBStateFile is the DB file containing metadata about the snapshots.
	SnapshotMetadataDBStateFile = StatePath(filepath.Join(SnapshotStateHome.String(), "snapshot_meta.db"))

	// WalletCLIStateHome is the folder containing the state of the wallet CLI.
	WalletCLIStateHome = StatePath("wallet-cli")

	// WalletCLILogsHome is the folder containing the logs of the wallet CLI.
	WalletCLILogsHome = StatePath(filepath.Join(WalletCLIStateHome.String(), "logs"))

	// WalletAppStateHome is the folder containing the state of the wallet
	// app.
	WalletAppStateHome = StatePath("wallet-app")

	// WalletAppLogsHome is the folder containing the logs of the wallet
	// app.
	WalletAppLogsHome = StatePath(filepath.Join(WalletAppStateHome.String(), "logs"))

	// WalletServiceStateHome is the folder containing the state of the node.
	WalletServiceStateHome = StatePath("wallet-service")

	// WalletServiceLogsHome is the folder containing the logs of the node.
	WalletServiceLogsHome = StatePath(filepath.Join(WalletServiceStateHome.String(), "logs"))
)
