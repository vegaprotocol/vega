# Vega

A decentralised trading platform that allows pseudo-anonymous trading of derivatives on a blockchain.

**Vega** provides the following core features:

<img align="right" src="https://vegaprotocol.io/img/concept_header.svg" height="170" style="padding: 0 10px 0 0;">

- Join a Vega network as a validator or non-concensus node. 
- Provision new markets on a network (coming soon)
- [Manage orders](#orders) (and [trade on a network](#trading))
- [Configure a node](#configuration) (and it's [APIs](#apis))
- Manage authentication with a network (coming soon) 
- Run scenario tests (coming soon)
- [Run benchmarks](#benchmarks) and test suites
- Support settlement in cryptocurrency (coming soon) 

[![Build Status](https://gitlab.com/vegaprotocol/trading-core/badges/master/pipeline.svgmaster)](https://gitlab.com/vegaprotocol/trading-core)
[![coverage](https://gitlab.com/gitlab-org/gitlab-ce/badges/master/coverage.svg)](https://gitlab.com/vega-protocol/trading-core/commits/master)

## Links

- For **updates**, see [CHANGELOG.md](CHANGELOG.md) for major updates, and
  [releases](https://gitlab.com/vega-protocol/trading-core/wikis/Release-notes) for a detailed version history.
- For **architecture**, please read the design [documentation](ARCHITECTURE.md) to learn about the design for the system and it's architecture.
- For **agile process**, please read the engineering [documentation](AGILE.md) or ask on #Engineering if you need further clarification.
- Please [open an issue](https://gitlab.com/vegaprotocol/trading-core/issues/new) if anything is missing or unclear in this documentation.


<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [APIs](#apis)
- [Provisioning](#provisioning)
- [Trading](#trading)
- [Benchmarks](#benchmarks)
- [Troubleshooting & debugging](#troubleshooting--debugging)

<!-- tocstop -->

</details>

## Installation

### Requirements


To install Vega from source, the following software is required:

* `Golang (Go) v1.11.5` - [Installation guide](https://golang.org/doc/install)
* `Tendermint v0.30` - [Installation guide](https://tendermint.com/docs/introduction/install.html)

### Tendermint ###

[Tendermint](https://tendermint.com/docs/introduction/what-is-tendermint.html) performs Byzantine Fault Tolerant (BFT) State Machine Replication (SMR) for arbitrary deterministic, finite state machines. Tendermint core is required for Vega nodes to communicate.

*We recommend downloading a pre-built core binary for your architecture rather than compiling from source, to save time. You can of course install from source if you wish, just make sure you grab the correct version.*

Once installed check the version, this should match the required version (above):

```
# verify tendermint
tendermint version
```

Next, initialise Tendermint core:

```
# initialize tendermint
tendermint init
```
Finally, start Tendermint:

```
# start creating blocks with Tendermint
tendermint node
```

Optionally, if you're running a multi node network - configure the Tendermint node settings by editing `genesis.json` and `config.toml`:

```
# Configure tendermint
cd ~/.tendermint/config
pico config.toml 
pico genesis.json
```

Tip: to clear and reset chain data (back to genesis block), run `tendermint unsafe_reset_all`

### Vega

To install or build Vega core, the source code is required. To check out the code, please follow these steps (on a *nix system):

```
mkdir -p ~/vega
git clone git@gitlab.com:vega-protocol/trading-core.git ~/vega
export GOMODULE111=on
go mod download
```

Tip: this project uses go module based dependency management, we recommend checking out the source outside of your Go src directory.


### Global

As a globally available command (installed in your Go path):

```
make install
```

### Local

As a single `binary` in your project:

```
make build
```

## Usage


Run a node:

```
vega node
```

Initialise a node:

```
vega init
```

Help for a node:

```
vega help
```

Version for a node:

```
vega version
```

## Configuration

Vega is initialised with a set of default configuration with the command `vega init`. There are [plenty of options](/config.toml) to configure it. To override any of the defaults edit your `config.toml` typically found in the `~/.vega` directory. Example:

```
[Matching]
  Level = 0
  ProRataMode = false
  LogPriceLevelsDebug = false
  LogRemovedOrdersDebug = false
```

## APIs

In order for clients to communicate with Vega nodes we expose a set of APIs and methods for reading and writing data. Note: Most writes will typically require interaction with the blockchain and require consensus. 

There are currently three protocols to communicate with the Vega APIs:

### GraphQL

[GraphQL](https://graphql.org/) is an open-source data query and manipulation language for APIs, and a runtime for fulfilling queries with existing data, originally developed at Facebook. The [Client UI](https://gitlab.com/vega-protocol/client) uses the GraphQL API to retrieve data including streaming of events.

The GraphQL [schema](./internal/api/endpoints/gql/schema.graphql) defines the interop with Vega. External clients will use this schema to communicate with Vega.

Queries can be tested using the GraphQL playground app which is bundled with a node. The default port (configurable) for the playground app is `3004` accessing this in a web browser will show a web app for testing custom queries, mutations and subscriptions. 


### gRPC

gRPC is an open source remote procedure call (RPC) system initially developed at Google. In Vega the gRPC API features streaming of events in addition to standard procedure calls.

The default port (configurable) for the gRPC API is `3005` and matches the [gRPC proto definition](./internal/api/grpc.proto).


### REST

REST provides a standard between computer systems on the web, making it easier for systems to communicate with each other. It is arguably simpler to work with than gRPC and GraphQL. In Vega the REST API is a reverse proxy to the gRPC API, however it does not support streaming.

The default port (configurable) for the REST API is `3003` and we use a reverse proxy to the gRPC API to deliver the REST API implementation. 

## Provisioning

The provisioning of new markets is **coming soon**. 

Vega supports a single fixed market with ID `BTC/DEC19` which can be passed to APIs as the field `Market` in protobuf/rest/graphql requests.

## Trading

When trading derivatives on Vega, traders send messages to place buy or sell `orders` on a `market`, these are known as `aggressive` orders. If these `orders` match one or more corresponding opposite `passive` buy or sell `orders` already on the `order book`, then a set of `trades` will be generated. For more detailed information on trading terminology please see the [trading and protocol glossary](https://gitlab.com/vega-protocol/product/wikis/Trading-and-Protocol-Glossary) or speak with @barney/@tamlyn.

There are several trading operations currently supported by Vega, these are as follows:

### Submit order



### Amend order



### Cancel order



## Benchmarks

TODO - @ashleyvega

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the vega logs and tendermint logs which are available on the console if running locally or by multilog if running on test networks. Default testnet location for log files:

* `/home/vega/log/vega/`
* `/home/vega/log/tendermint/`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.

Debugging the application locally is also possible with [Delve](https://github.com/go-delve/delve).
