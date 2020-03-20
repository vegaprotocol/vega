# Vega

Version 0.16.0.

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
- For **updates**, see the [Change log](CHANGELOG.md) for major updates, and
  [releases](https://gitlab.com/vega-protocol/trading-core/wikis/Release-notes) for a detailed version history.
- For **architecture**, please read the [design documentation](design/index.md) to learn about the design for the system and its architecture.
- For **agile process**, please read the [engineering documentation](AGILE.md) or ask on Slack channel `#engineering` if you need further clarification.
- Please [open an issue](https://gitlab.com/vegaprotocol/trading-core/issues/new) if anything is missing or unclear in this documentation.

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

Vega is initialised with a set of default configuration with the command `vega init`. There are [plenty of options](/config.toml) to configure it. To override any of the defaults edit your `config.toml` typically found in the `~/.vega` directory. Example:

```toml
[Matching]
  Level = 0
  ProRataMode = false
  LogPriceLevelsDebug = false
  LogRemovedOrdersDebug = false
```

## APIs

In order for clients to communicate with Vega nodes, we expose a set of APIs and methods for reading and writing data. Note: Most writes will typically require interaction with the blockchain and require consensus.

There are currently three protocols to communicate with the Vega APIs:

### gRPC

gRPC is an open source remote procedure call (RPC) system initially developed at Google. In Vega the gRPC API features streaming of events in addition to standard procedure calls.

The default port (configurable) for the gRPC API is 3002 and matches the [gRPC protobuf definition](proto/api/trading.proto).

### GraphQL

[GraphQL](https://graphql.org/) is an open-source data query and manipulation language for APIs, and a runtime for fulfilling queries with existing data, originally developed at Facebook. The [Client UI](https://gitlab.com/vega-protocol/client) uses the GraphQL API to retrieve data including streaming of events.

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

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.

Debugging the application locally is also possible with [Delve](https://github.com/go-delve/delve).
