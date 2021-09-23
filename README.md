# Vega

Version 0.42.0.

A decentralised trading platform that allows pseudo-anonymous trading of derivatives on a blockchain.

**Vega** provides the following core features:

- Join a Vega network as a validator or non-consensus node.
- [Governance](./governance/README.md) - proposing and voting for new markets
- A [matching engine](./matching/README.md)
- [Configure a node](#configuration) (and its [APIs](#apis))
- Manage authentication with a network
- [Run scenario tests](./integration/README.md)
- Support settlement in cryptocurrency (coming soon)

Additional services that are in this repo, but run separately:
- [Wallet](./wallet/README.md) can be used provide key management for users.

## Links

- For **new developers**, see [Getting Started](GETTING_STARTED.md).
- For **updates**, see the [Change log](CHANGELOG.md) for major updates.
- For **architecture**, please read the [documentation](docs/index.md) to learn about the design for the system and its architecture.
- Please [open an issue](https://github.com/vegaprotocol/vega/issues/new) if anything is missing or unclear in this documentation.

<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Installation](#installation)
- [Configuration](#configuration)
- [APIs](#apis)
- [Provisioning](#provisioning)
- [Benchmarks](#benchmarks)
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

| Environment variables | Unix             | macOS                           | Windows                |
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

## APIs

In order for clients to communicate with Vega nodes, we expose a set of APIs and methods for reading and writing data. Note: Most writes will typically require interaction with the blockchain and require consensus.

There are currently three protocols to communicate with the Vega APIs:

### gRPC

gRPC is an open source remote procedure call (RPC) system initially developed at Google. In Vega the gRPC API features streaming of events in addition to standard procedure calls.

The default port (configurable) for the gRPC API is 3002 and matches the [gRPC protobuf definition](proto/api/trading.proto).

### GraphQL

[GraphQL](https://graphql.org/) is an open-source data query and manipulation language for APIs, and a runtime for fulfilling queries with existing data, originally developed at Facebook. The [Console](https://github.com/vegaprotocol/console) uses the GraphQL API to retrieve data including streaming of events.

The GraphQL API is defined by a [schema](gateway/graphql/schema.graphql). External clients will use this schema to communicate with Vega.

Queries can be tested using the GraphQL playground app which is bundled with a node. The default port (configurable) for the playground app is `3004` accessing this in a web browser will show a web app for testing custom queries, mutations and subscriptions.

### REST

REST provides a standard between computer systems on the web, making it easier for systems to communicate with each other. It is arguably simpler to work with than gRPC and GraphQL. In Vega the REST API is a reverse proxy to the gRPC API, however it does not support streaming.

The default port (configurable) for the REST API is `3003` and we use a reverse proxy to the gRPC API to deliver the REST API implementation.

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
