# Vega

Version 0.45.3

A decentralised trading platform that allows pseudo-anonymous trading of derivatives on a blockchain.

**Vega** provides the following core features:

- Join a Vega network as a validator or non-consensus node.
- [Governance](./governance/README.md) - proposing and voting for new markets
- A [matching engine](./matching/README.md)
- [Configure a node](#configuration) (and its [APIs](#apis))
- Manage authentication with a network
- [Run scenario tests](./integration/README.md)
- Support settlement in cryptocurrency (coming soon)
## Links

- For **new developers**, see [Getting Started](GETTING_STARTED.md).
- For **updates**, see the [Change log](CHANGELOG.md) for major updates.
- For **architecture**, please read the [documentation](docs/index.md) to learn about the design for the system and its architecture.
- Please [open an issue](https://github.com/vegaprotocol/vega/issues/new) if anything is missing or unclear in this documentation.

<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Vega](#vega)
  - [Links](#links)
  - [Installation](#installation)
  - [Configuration](#configuration)
    - [Files location](#files-location)
  - [Vega node wallets](#vega-node-wallets)
    - [Using Ethereum Clef wallet](#using-ethereum-clef-wallet)
      - [Automatic approvals](#automatic-approvals)
      - [Importing and generation account](#importing-and-generation-account)
  - [API](#api)
  - [Provisioning](#provisioning)
  - [Troubleshooting & debugging](#troubleshooting--debugging)

<!-- tocstop -->

</details>

## Installation

To install `trading-core` and `tendermint`, see [Getting Started](GETTING_STARTED.md).

## Configuration

Vega is initialised with a set of default configuration with the command `vega init`. To override any of the defaults, edit your `config.toml`.

**Example**

```toml
[Matching]
Level = 0
ProRataMode = false
LogPriceLevelsDebug = false
LogRemovedOrdersDebug = false
```

Vega requires a set of wallets for the internal or external chain it's dealing with.

The node wallets can be accessed using the `nodewallet` subcommand, these node wallets are initialized / accessed using a passphrase that needs to be specified when initializing Vega:

```sh
vega init --nodewallet-passphrase-file "my-passphrase-file.txt"
```

### Files location

| Environment variables | Unix             | MacOS                           | Windows                |
| :-------------------- | :----------------| :------------------------------ | :--------------------- |
| `XDG_DATA_HOME`       | `~/.local/share` | `~/Library/Application Support` | `%LOCALAPPDATA%`       |
| `XDG_CONFIG_HOME`     | `~/.config`      | `~/Library/Application Support` | `%LOCALAPPDATA%`       |
| `XDG_STATE_HOME`      | `~/.local/state` | `~/Library/Application Support` | `%LOCALAPPDATA%`       |
| `XDG_CACHE_HOME`      | `~/.cache`       | `~/Library/Caches`              | `%LOCALAPPDATA%\cache` |

You can override these environment variables, however, bear in mind it will apply system-wide.

If you don't want to rely on the default XDG paths, you can use the `--home` flag on the command-line.

## Vega node wallets

A Vega node needs to connect to other blockchain for various operation:
- validate transaction happened on foreign chains
- verify presence of assets
- sign transaction to be verified on foreign blockchain
- and more...

In order to do these different action, the Vega node needs to access these chains using their native wallet. To do so the vega command line provide a command line tool:
`vega nodewallet` allowing users to import foreign blockchain wallets credentials, so they can be used at runtime.

For more details on how to use the Vega node wallets run:
```
vega nodewallet --help
```

### Using Ethereum Clef wallet

#### Automatic approvals

Given that Clef requires manually approving all RPC API calls, it is mandatory to setup
[custom rules](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/rules.md#rules) for automatic approvals. Vega requires at least `ApproveListing` and `ApproveSignData` rules to be automatically approved.

Example of simple rule set JavaScript file with approvals required by Vega:
```js
function ApproveListing() {
  return "Approve"
}

function ApproveSignData() {
  return "Approve"
}
```

Clef also allows more refined rules for signing. For example approves signs from `ipc` socket:
```js
function ApproveSignData(req) {
  if (req.metadata.scheme == "ipc") {
    return "Approve"
  }
}
```

Please refer to Clef [rules docs](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/rules.md#rules) for more information.

#### Importing and generation account

As of today, Clef does not allow to generate a new account for other back end storages than a local Key Store. Therefore it is preferable to create a new account on the back end of choice and import it to Vega through node wallet CLI.

Example of import:
```sh
vega nodewallet import --chain=ethereum --eth.clef-address=http://clef-address:port
```

## API

Prior to version 0.40.0, Vega Core hosted API endpoints for clients. The majority of this has since migrated to the [data-node](https://github.com/vegaprotocol/data-node).
## Provisioning

The proposal and creation of new markets is handled by the [Governance engine](./governance/README.md).

Vega supports a single fixed market with ID `BTC/DEC20` which can be passed to APIs as the field `Market` in protobuf / REST / GraphQL requests.


Cancellations typically go via consensus so the `OrderResponse` will only indicate that the message was accepted and sent out onto the blockchain to be included in a block. It could be rejected at a later stage of processing.


## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the Vega and Tendermint logs which are available on the console if running locally or by journal plus syslog if running on test networks. Default location for log files:

* `/var/log/vega.log`
* `/var/log/tendermint.log`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `"Debug"` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.

Debugging the application locally is also possible with [Delve](./DEBUG_WITH_DLV.md).
