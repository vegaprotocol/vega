# Data node

Version 0.46.0.

A service exposing read only APIs built on top of [Vega](https://github.com/vegaprotocol/vega) platform.

**Data node** provides the following core features:

- Consume all events from Vega core
- Aggregates received events and stores the aggregated data
- Serves stored data via [APIs](#apis)
- Allows advanced configuration [Configure a node](#configuration)

## Links

- For **new developers**, see [Getting Started](GETTING_STARTED.md).
- For **updates**, see the [Change log](CHANGELOG.md) for major updates.
- For **architecture**, please read the [documentation](docs/index.md) to learn about the design for the system and its architecture.
- Please [open an issue](https://github.com/vegaprotocol/data-node/issues/new) if anything is missing or unclear in this documentation.

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

To install see [Getting Started](GETTING_STARTED.md).

## Configuration

Data node is initialised with a set of default configuration with the command `data-node init`. To override any of the defaults edit your `config.toml` typically found in the `~/.data-node` directory. Example:

```toml
[Matching]
  Level = 0
  ProRataMode = false
  LogPriceLevelsDebug = false
  LogRemovedOrdersDebug = false
```

## Vega core streaming

Data requires an instance of Vega core node for it's meaningful function. Please see [Vega Getting Started](https://github.com/vegaprotocol/vega/blob/develop/GETTING_STARTED.md).
The data node will listen on default port `3002` for incoming connections from Vega core node.

## APIs

In order for clients to communicate with data nodes, we expose a set of APIs and methods for reading data.

There are currently three protocols to communicate with the data node APIs:

### gRPC

gRPC is an open source remote procedure call (RPC) system initially developed at Google. In data node the gRPC API features streaming of events in addition to standard procedure calls.

The default port (configurable) for the gRPC API is `3007` and matches the [gRPC protobuf definition](https://github.com/vegaprotocol/protos).

### GraphQL

[GraphQL](https://graphql.org/) is an open-source data query and manipulation language for APIs, and a runtime for fulfilling queries with existing data, originally developed at Facebook. The [Console](https://github.com/vegaprotocol/console) uses the GraphQL API to retrieve data including streaming of events.

The GraphQL API is defined by a [schema](gateway/graphql/schema.graphql). External clients will use this schema to communicate with Vega.

Queries can be tested using the GraphQL playground app which is bundled with a node. The default port (configurable) for the playground app is `3008` accessing this in a web browser will show a web app for testing custom queries, mutations and subscriptions.

### REST

REST provides a standard between computer systems on the web, making it easier for systems to communicate with each other. It is arguably simpler to work with than gRPC and GraphQL. In Vega the REST API is a reverse proxy to the gRPC API, however it does not support streaming.

The default port (configurable) for the REST API is `3009` and we use a reverse proxy to the gRPC API to deliver the REST API implementation.

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the Vega and Tendermint logs which are available on the console if running locally or by journal plus syslog if running on test networks. Default location for log files:

* `/var/log/vega.log`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.
