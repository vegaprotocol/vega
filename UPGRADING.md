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

## Configuration changes

### Vega

The vega configuration file can be found under `$VEGA_HOME/config/node/config.toml`

#### Settings added in v0.67.0

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

#### Settings removed in v0.67.0

**_UlimitNOFile_** - previously used to increase the amount of fd allowed to be created by the node, it was required for the internal use of badger which have been removed.

**_Admin.Server.Enabled_** - previously used to disable the admin server, this is not an option anymore as this is required for protocol upgrades.

**_Blockchain.Tendermint.ClientAddr_**, **_Blockchain.Tendermint.ClientEndpoint_**, **_Blockchain.Tendermint.ServerPort_**, **_Blockchain.Tendermint.ServerAddr_** - vega is now using a builtin tendermint application, there's no need to setup configuration with an external tendermint node.

**_[Monitoring] section_** - this section have been removed.

**_[NodeWallet.ETH]_** - This have been removed from the _[NodeWallet]_ section to be set into it's own _[Ethereum]_ section.


### Tendermint

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
