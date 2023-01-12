Upgrading from v0.53.0 to v0.67.0
=================================

In the way to the v0.67.0 release, most vega code have been open sourced. At the same time, the code for the wallet ([previously](https://github.com/vegaprotocol/vegawallet)) and the datanode ([previously](https://github.com/vegaprotocol/data-node)) have been imported in this repository.

The binaries are still available as standalone and can be downloaded through the github release [page](https://github.com/vegaprotocol/vega/releases) but are also available under the vega toolchain:
```
vega datanode --help
vega wallet --help
```

The vega core node is also now a builtin tendermint application. This means that it's not necessary anymore to run tendermint separately. Most tendermint commands used to managed a tendermint chain are also available under the vega toolchain:
```
vega tendermint --help
```

# Configuration changes

## Vega

The vega configuration file can be found under `$VEGA_HOME/config/node/config.toml`

### Settings added in v0.67.0

**_MaxMemoryPercent_** - A value to control the maximum amount the vega node will use. The accept range of value is 1-100, 100 basically removing any memory usage restriction. By default set to 33 when initialising a full node (accounting for a possible datanode running as well on the same hardware) and 100 when initialising a validator.

Usage example:
```Toml
# set the memory usage to 50% max of the available resources on the hardware
MaxMemoryPercent = 50
```

**_[Ethereum] section_** - This whole secton have been added in order to setup the configuration of the ethereum node the validators are using to validate events on the ethereum chain, it's required to set it for a validator node, unused when running a non validator node.

**note: The validator nodes require to connect against a ethereum archive node**

Usage example:
```Toml
[Ethereum]
 # control the log level of this package
 Level = "Info"
 # The address of the ethereum node RPC endpoint
 RPCEndpoint = "http://some_rpc_endpoint"
 RetryDelay = "15s"
```

**_EvtForward.Ethereum.PollEventRetryDuration_** - Configure how often the ethereum event source will try to find new activity on the ethereum bridge.

Usage Example:
```Toml
[EvtForward]
 [EvtForward.Ethereum]
  PollEventRetryDuration = "20s"
```

**_Snapshot.StartHeight_** - this parameter already existed but it's default has changed to `-1`, we recommend you set it to this value as it set the node to restart from the last local snapshot

Usage Example:
```Toml
[Snapshot]
 StartHeight = -1
```

### Settings removed in v0.67.0

**_UlimitNOFile_** - previously used to increase the amount of fd allowed to be created by the node, it was required for the internal use of badger which have been removed.

**_Admin.Server.Enabled_** - previously used to disable the admin server, this is not an option anymore as this is required for protocol upgrades.

**_Blockchain.Tendermint.ClientAddr_**, **_Blockchain.Tendermint.ClientEndpoint_**, **_Blockchain.Tendermint.ServerPort_**, **_Blockchain.Tendermint.ServerAddr_** - vega is now using a builtin tendermint application, there's no need to setup configuration with an external tendermint node.

**_[Monitoring] section_** - this section have been removed.

**_[NodeWallet.ETH]_** - This have been removed from the _[NodeWallet]_ section to be set into it's own _[Ethereum]_ section.


## Tendermint

Here's a list of settings from the tendermint configuration that needs to be set so vega operate properly. You can find the tendermint configuration under `$TENDERMINT_HOME/config/config.toml`. Others can be kept to the defaults.

```Toml
[p2p]
# Maximum size of a message packet payload, in bytes
max_packet_msg_payload_size = 16384

[mempool]
# Mempool version to use:
#   1) "v0" - (default) FIFO mempool.
#   2) "v1" - prioritized mempool.
version = "v1"
# Maximum number of transactions in the mempool
size = 10000
# Size of the cache (used to filter transactions we saw earlier) in transactions
cache_size = 20000

[consensus]
# How long we wait after committing a block, before starting on the new
# height (this gives us a chance to receive some more precommits, even
# though we already have +2/3).
timeout_commit = "0s"
# Make progress as soon as we have all the precommits (as if TimeoutCommit = 0)
skip_timeout_commit = true
# EmptyBlocks mode and possible interval between empty blocks
create_empty_blocks = true
create_empty_blocks_interval = "1s"
```

## Data node

The data node configuration file can be found under `$DATANODE_HOME/config/data-node/config.toml`

### Settings added in v0.67.0

**_MaxMemoryPercent_** - A value to control the maximum amount the vega node will use. The accept range of value is 1-100 the default value is 33 assuming that the datanode is running on the same host as the vega core node and postgres.

Usage example:
```Toml
# set the memory usage to 50% max of the available resources on the hardware
MaxMemoryPercent = 50
```

**_AutoInitialiseFromDeHistory_** - Should the datanode be bootstrapping it's state from other datanodes in the network.

Usage example:
```Toml
AutoInitialiseFromDeCentralisedHistory = false
```
**_ChainID_** - The chain ID of the current vega mainnet, this is being set automatically when running `init` for the first time.

Usage example:
```Toml
ChainID = "vega-mainnet-0009"
```

**_[Admin] section_** - The configuration for the admin local API, this is generate automatically when running `init` for the first time.

Usage example:
```Toml
[Admin]
  Level = "Info"
  [Admin.Server]
    SocketPath = "/var/folders/l7/lq57j66j6hjdllwffykpqf_h0000gn/T/datanode.sock"
    HTTPPath = "/datanode/rpc"
```

**_SQLStore.WipeOnStartup_** - This setting would delete the postgres database on every start, clearing up all state, we recommend to set this to false

Usage example:
```Toml
[SQLStore]
 WipeOnStartup = false
```

**_SQLStore.ConnectionRetryConfig_**, **_SQLStore.LogRotationConfig_** - Advanced configuration for the postgres connector. We recommend you use the default setting created when running the `init` command.

**_Gateway.MaxSubscriptionPerClient_** - The maximum amount of Graphql subsciption allowed per client connection.

Usage example:
```Toml
[Gateway]
 MaxSubscriptionPerClient = 100
```

**_Gateway.GraphQL.Endpoint_** - The endpoint serving the GraphQL API, the default is set to the standard endpoint for GraphQL APIs.

```Toml
[Gateway]
 [Gateway.GraphQL]
  Endpoint = "/graphql"
```

**_Broker.UseBufferedEventSource_** - The broker is the connection between the vega core node and data node, this connection needs to be stable at any time to ensure the data node can reconcile all the state out of the vega events. This setting allow the datanode to use a buffer when it's not able to consume events as fast as the vega core node produce them, we recommend to set this to true

Usage example:
```Toml
[Broker]
 UseBufferedEventSource = true
```

**_[Broker.BufferedEventSourceConfig] section_** - This section configure the buffered event source mentioned previously. We recommend to use the default from the `init` command.

**_[NetworkHistory] section_** - This configure the p2p history of the data nodes on the network. We recommend you use the default configuration create when running the `init` command.

### Settings removed in v0.67.0

**_UlimitNOFile_** - previously used to increase the amount of fd allowed to be created by the node, it was required for the internal use of badger which have been removed.

**_API.ExposeLegacyAPI_**, **_API.LegacyAPIPortOffset_** - The legacy API have been fully removed in the new version, these fields are un-necessary.

# Command line changes

## Vega

The `node` subcommand have been deprecated in favour of `start`, you should start using the new command as the previous one will be fully removed in a future release.

The `tm` subcommand have been deprecated in favour of `tendermint`, you should start using the new command as the previous one will be fully removed in a future release. Also, vega being now a tendermint builtin application, some of the commands exposed by the subcommand have also been removed like the `node` subcommand.

The `init` and `start` command now takes an optional `--tendermint-home` used to specify the home of the tendermint configuration and state. If ignored, the default tendermint home is used (`$HOME/.tendermint`).

The `init` command also takes an optional `--no-tendermint` which will avoid creating the tendermint configuration.

The `protocol_upgrade_proposal` subcommand has been introduce, this is used by validator nodes to share on chain their intent to upgrade to a newer version of the protocol.

The vega toolchain now also expose some of the other programs under the following subcommands:
- `tools`: Some tools use for introspection of the vega chain
- `wallet`: the vega wallet
- `datanode`: the vega data node
- `blockexplorer`: the api use by the block explorer

Refer to their documentation for more information or use the standard `--help` for more details

For extended help about the vega toolchain run:
```Shell
vega --help
```

## Datanode

The `node` subcommand have been deprecated in favour of `start`, you should start using the new command as the previous one will be fully removed in a future release.

The `init` command now requires the chain-id of the current vega network. For example if we were to initial a datanode for the current mainnet, we would run the following command:
```Shell
vega datanode init --home="$DATANODE_HOME" "vega-mainnet-0009"
```

The new `network-history` subcommand provide tools to manage the history of the data node which is shared accross the network through IPFS.

The new `last-block` command returns the last block processed by the data node.
