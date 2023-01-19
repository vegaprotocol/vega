Upgrading from v0.53.0 to v0.67.1
=================================

Read through for a list of the major changes between versions 0.53.0 and 0.67.1, covering:
* [Repository changes](#repository-changes)
* [Configuration changes](#configuration-changes)
* [Command line changes](#command-line-changes)

# Repository changes

Along the way to the v0.67 release, most Vega code has been made public. In addition, the code for the Vega Wallet ([previously](https://github.com/vegaprotocol/vegawallet)) and the data node ([previously](https://github.com/vegaprotocol/data-node)) have been imported in this repository, and both of the previous repositories are now unmaintained.

The binaries are still available as standalone files in the Vega repo and can be downloaded through the GitHub release [page](https://github.com/vegaprotocol/vega/releases). When downloading the binaries, be sure to choose the compatible software version to that of the network. However, they are also available under the vega toolchain:
```
vega datanode --help
vega wallet --help
```

The Vega core node is now a built-in Tendermint application. This means that it's no longer required (and not recommended) to run Tendermint separately. Most Tendermint commands used to manage a Tendermint chain are also available under the Vega toolchain:
```
vega tendermint --help
```

## New service
With version 0.67, `vegavisor` was introduced to help facilitate protocol upgrades. This tool is not required to run the node but recommended in order to ease software upgrades when they're expected by the network. Read more about [Vega Visor](https://github.com/vegaprotocol/vega/blob/develop/visor/readme.md).

# Configuration changes

## Vega

The Vega configuration file can be found in `$VEGA_HOME/config/node/config.toml`.

### Settings added in v0.67

**_MaxMemoryPercent_** - A value to control the maximum amount of memory the Vega node will use. The accept range of value is 1-100: 100 basically removes any memory usage restriction. The default is set to 33 when initialising a full node (accounting for a possible data node running as well on the same hardware) and 100 when initialising a validator.

Usage example:
```Toml
# set the memory usage to 50% max of the available resources on the hardware
MaxMemoryPercent = 50
```

**_[Ethereum] section_** - This whole section has been added to set up the configuration of the Ethereum node that validators use to validate events on the Ethereum chain. It's required to set it for a validator node, and unused when running a non-validator node.

**note: Validator nodes are required to connect to an Ethereum archive node.**

Usage example:
```Toml
[Ethereum]
 # control the log level of this package
 Level = "Info"
 # The address of the ethereum node RPC endpoint
 RPCEndpoint = "http://some_rpc_endpoint"
 RetryDelay = "15s"
```

**_EvtForward.Ethereum.PollEventRetryDuration_** - Configure how often the Ethereum event source will try to find new activity on the Ethereum bridge.

Usage Example:
```Toml
[EvtForward]
 [EvtForward.Ethereum]
  PollEventRetryDuration = "20s"
```

**_Snapshot.StartHeight_** - This parameter already existed, but the default has changed to `-1`. We recommend setting it to this value as it makes the node restart from the last local snapshot.

Usage Example:
```Toml
[Snapshot]
 StartHeight = -1
```

### Settings removed in v0.67

**_UlimitNOFile_** - Previously used to increase the number of FD created by the node. It was required for internal use of Badger, which has been removed.

**_Admin.Server.Enabled_** - Previously used to disable the admin server. This is not an option anymore as it is required for protocol upgrades.

**_Blockchain.Tendermint.ClientAddr_**, **_Blockchain.Tendermint.ClientEndpoint_**, **_Blockchain.Tendermint.ServerPort_**, **_Blockchain.Tendermint.ServerAddr_** - As Vega now uses a built-in Tendermint application, there's no need to set up configuration with an external Tendermint node.

**_[Monitoring] section_** - This section has been removed.

**_[NodeWallet.ETH]_** - This has been removed from the _[NodeWallet]_ section, and is now set in its own _[Ethereum]_ section.

## Tendermint

The Tendermint configuration can be found in `$TENDERMINT_HOME/config/config.toml`.

Below is a list of Tendermint configuration settings that need to be set so Vega operates correctly. Others can be kept at the defaults.

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

The data node configuration file can be found in `$DATANODE_HOME/config/data-node/config.toml`.

### Settings added in v0.67

**_MaxMemoryPercent_** - A value to control the maximum amount of memory the Vega node will use. The accepted range of values is 1-100. The default value is 33, assuming that the data node is running on the same host as the Vega core node and Postgres.

Usage example:
```Toml
# set the memory usage to 50% max of the available resources on the hardware
MaxMemoryPercent = 50
```

**_AutoInitialiseFromNetworkHistory_** - Used if the data node is bootstrapping its state from other data nodes in the network.

Usage example:
```Toml
AutoInitialiseFromNetworkHistory = false
```
**_ChainID_** - The chain ID of the current Vega mainnet. This is set automatically when running `init` for the first time.

Usage example:
```Toml
ChainID = "vega-mainnet-0009"
```

**_[Admin] section_** - The configuration for the admin's local API. This is generated automatically when running `init` for the first time.

Usage example:
```Toml
[Admin]
  Level = "Info"
  [Admin.Server]
    SocketPath = "/var/folders/l7/lq57j66j6hjdllwffykpqf_h0000gn/T/datanode.sock"
    HTTPPath = "/datanode/rpc"
```

**_SQLStore.WipeOnStartup_** - This setting would delete the Postgres database on every restart, clearing all state. We recommend setting this to `false`.

Usage example:
```Toml
[SQLStore]
 WipeOnStartup = false
```

**_SQLStore.ConnectionRetryConfig_**, **_SQLStore.LogRotationConfig_** - Advanced configuration for the Postgres connector. We recommend using the default setting created when running the `init` command.

**_Gateway.MaxSubscriptionPerClient_** - The maximum number of GraphQL subscriptions allowed per client connection. The default is set to 250.

Usage example:
```Toml
[Gateway]
 MaxSubscriptionPerClient = 100
```

**_Gateway.GraphQL.Endpoint_** - The endpoint serving the GraphQL API. The default is set to the standard endpoint for GraphQL APIs.

```Toml
[Gateway]
 [Gateway.GraphQL]
  Endpoint = "/graphql"
```

**_Broker.UseBufferedEventSource_** - The broker is the connection between the Vega core node and data node. This connection needs to be stable at all times to ensure the data node can reconcile all the state from the Vega events. This setting allows the data node to use a buffer when it's not able to consume events as fast as the Vega core node produces them. We recommend setting this to `true`.

Usage example:
```Toml
[Broker]
 UseBufferedEventSource = true
```

**_[Broker.BufferedEventSourceConfig] section_** - This section configures the buffered event source mentioned previously. We recommend using the default from the `init` command.

**_[NetworkHistory] section_** - This configures the network history settings for a data node. We recommend using the default configuration created when running the `init` command.

### Settings removed in v0.67

**_UlimitNOFile_** - Previously used to increase the number of FD created by the node. It was required for the internal use of Badger, which has been removed.

**_API.ExposeLegacyAPI_**, **_API.LegacyAPIPortOffset_** - The legacy API has been fully removed in the new version, so these fields are unnecessary.

### Settings moved in v0.67

Previously it was only possible to enable HTTPS support on the GraphQL API gateway. In 0.67.1 we also added support for enabling HTTPS on the REST gateway. As a result, the HTTPS settings have moved from the `[Gateway.GraphQL]` to the  `[Gateway]` section.

For example, if your previous settings were

```toml
[Gateway]
  ...
  [Gateway.GraphQL]
    HTTPSEnabled = true
    AutoCertDomain = "my.domain.com"
    CertificateFile = ""
    KeyFile = ""
    ...
```
They must now become
```toml
[Gateway]
  ...
  HTTPSEnabled = true
  AutoCertDomain = "my.domain.com"
  CertificateFile = ""
  KeyFile = ""
  [Gateway.GraphQL]
    ...
```
Please note, that for `autocert` to work then either the GraphQL or REST endpoints *must* be reachable on the internet at `my.domain.com:443` (this is a hard requirement from LetsEncrypt). You could forward the port with a firewall rule or proxy, or simply specify 443 as the port for one of them in the `[Gateway.GraphQL]` or `[Gateway.Rest]` config sections.

If that is not possible, you will need to use [other means](https://letsencrypt.org/getting-started/) to generate and provide a signed certificate to datanode. Specify the path to your certificate & private key in the `CertificateFile` and `KeyFile` options.

Please note, you may either use `autocert` to manage your certificates *or* provide certificate files. Datanode will refuse to start if you both enable `autocert` by specifying an `AutoCertDomain` and also specify a certificate via `CertificateFile`


# Command line changes

## Vega

#### Deprecations

`node` subcommand has been deprecated in favour of `start`. You should use the new command as the previous one will be fully removed in a future release.

`tm` subcommand has been deprecated in favour of `tendermint`. You should use the new command as the previous one will be fully removed in a future release. Also, as Vega now has Tendermint as a built-in application, some of the commands exposed by the subcommand have also been removed, such as the `node` subcommand.

#### Additions

`init` and `start` now take an optional `--tendermint-home` used to specify the home of the Tendermint configuration and state. If ignored, the default Tendermint home is used (`$HOME/.tendermint`).

`init` also takes an optional `--no-tendermint` which will skip creating the Tendermint configuration.

`protocol_upgrade_proposal` subcommand has been introduced. This is used by validator nodes to share on-chain their intent to upgrade to a newer version of the protocol.

The Vega toolchain now also exposes some of the other programs under the following subcommands:
- `tools`: Tools used for interrogating the Vega chain
- `wallet`: The Vega Wallet
- `datanode`: The Vega data node
- `blockexplorer`: The API used by the block explorer

Refer to the documentation for more information, or use the standard `--help` flag for more details.

For extended help about the Vega toolchain run:
```Shell
vega --help
```

### Vega Wallet

The Vega Wallet has had several commands changed. Refer to the Vega Wallet `--help` to see the latest commands and guidance on how to use them.  

## Data node

#### Deprecations

`node` subcommand has been deprecated in favour of `start`. You should use the new command as the previous one will be fully removed in a future release.

#### Additions

`init` now requires the chain-id of the current Vega network. For example, if you were to initialise a data node for the current mainnet, you would run the following command:

```Shell
vega datanode init --home="$DATANODE_HOME" "vega-mainnet-0009"
```

`network-history`, a new subcommand, provides tools to manage the network history segments saved by the data node, which are shared across the network through IPFS.

`last-block`, a new command, returns the last block processed by the data node.
