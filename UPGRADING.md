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

_MaxMemoryPercent_ - A value to control the maximum amount the vega node will use. The accept range of value is 1-100, 100 basically removing any memory usage restriction. By default set to 33 when initialising a full node (accounting for a possible datanode running as well on the same hardware) and 100 when initialising a validator.

Usage example:
```
# set the memory usage to 50% max of the available resources on the hardware
MaxMemoryPercent = 50
```

_[Ethereum] section_ - This whole secton have been added in order to setup the configuration of the ethereum node the validators are using to validate events on the ethereum chain, it's required to set it for a validator node, unused when running a non validator node.

note: *The validator nodes require to connect against a ethereum archive node*

Usage example:
```
[Ethereum]
 # control the log level of this package
 Level = "Info"
 # The address of the ethereum node RPC endpoint
 RPCEndpoint = "http://some_rpc_endpoint"
 RetryDelay = "15s"
```

_EvtForward.Ethereum.PollEventRetryDuration_ - Configure how often the ethereum event source will try to find new activity on the ethereum bridge.

Usage Example:
```
[EvtForward]
 [EvtForward.Ethereum]
  PollEventRetryDuration = "20s"
```

_Snapshot.StartHeight_ - this parameter already existed but it's default has changed to `-1`, we recommend you set it to this value as it set the node to restart from the last local snapshot

Usage Example:
```
[Snapshot]
 StartHeight = -1
```


#### Settings removed in v0.67.0

_UlimitNOFile_ - previously used to increase the amount of fd allowed to be created by the node, it was required for the internal use of badger which have been removed.
_Admin.Server.Enabled_ - previously used to disable the admin server, this is not an option anymore as this is required for protocol upgrades.

_Blockchain.Tendermint.ClientAddr_, _Blockchain.Tendermint.ClientEndpoint_, _Blockchain.Tendermint.ServerPort_, _Blockchain.Tendermint.ServerAddr_ - vega is now using a builtin tendermint application, there's no need to setup configuration with an external tendermint node.

_[Monitoring] section_ - this section have been removed.

_[NodeWallet.ETH]_ - This have been removed from the _[NodeWallet]_ section to be set into it's owne [Ethereum] section.
