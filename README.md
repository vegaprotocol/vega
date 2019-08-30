# Vega

Version 0.9.0.

A decentralised trading platform that allows pseudo-anonymous trading of derivatives on a blockchain.

**Vega** provides the following core features:

<img align="right" src="https://vega.xyz/img/concept_header.svg" height="150" style="padding: 0 10px 0 0;">

- Join a Vega network as a validator or non-consensus node.
- [Provision](#provisioning) new markets on a network (coming soon)
- [Manage orders](#trading) (and [trade on a network](#trading))
- [Configure a node](#configuration) (and its [APIs](#apis))
- Manage authentication with a network (coming soon)
- Run scenario tests (coming soon)
- [Run benchmarks](#benchmarks) and test suites
- Support settlement in cryptocurrency (coming soon)

## Links

- For **new starters**, see [GETTING_STARTED.md](doc/GETTING_STARTED.md).
- For **updates**, see [CHANGELOG.md](CHANGELOG.md) for major updates, and
  [releases](https://gitlab.com/vega-protocol/trading-core/wikis/Release-notes) for a detailed version history.
- For **architecture**, please read the [design documentation](ARCHITECTURE.md) to learn about the design for the system and its architecture.
- For **agile process**, please read the [engineering documentation](AGILE.md) or ask on Slack channel `#Engineering` if you need further clarification.
- Please [open an issue](https://gitlab.com/vegaprotocol/trading-core/issues/new) if anything is missing or unclear in this documentation.

<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Installation](#installation)
- [Configuration](#configuration)
- [APIs](#apis)
- [Provisioning](#provisioning)
- [Trading](#trading)
- [Benchmarks](#benchmarks)
- [Troubleshooting & debugging](#troubleshooting--debugging)

<!-- tocstop -->

</details>

## Installation

To install `trading-core` and `tendermint`, see [GETTING_STARTED.md](doc/GETTING_STARTED.md).

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

Vega supports a single fixed market with ID `BTC/DEC19` which can be passed to APIs as the field `Market` in protobuf / REST / GraphQL requests.

## Trading

When trading derivatives on Vega, traders send messages to place buy or sell `orders` on a `market`, these are known as `aggressive` orders. If these `orders` match one or more corresponding opposite `passive` buy or sell `orders` already on the `order book`, then a set of `trades` will be generated. For more detailed information on trading terminology please see the [trading and protocol glossary](https://gitlab.com/vega-protocol/product/wikis/Trading-and-Protocol-Glossary) or speak with @barnabee (Slack:`@barney`) or @tamlyn10 (Slack:`@tamlyn`).

There are several trading operations currently supported by Vega, using the gRPC API for examples, these are as follows:

### Submit order

```
rpc CreateOrder(vega.Order) returns (OrderResponse);
```

To submit a new order to the network, a caller can submit a protobuf `order` message and receive an `OrderResponse` from the API. In the following example a trader wishes to `buy` a total of `500` contracts at price `100` on market ID `BTC/DEC19`:

**Request**

```
message Order {
	string id = "";
    string market = "BTC/DEC19";
    string party = "goldman";
    Side side = Buy;
    uint64 price = 100;
    uint64 size = 500;
    uint64 remaining = 500;
    Type type = GTC;
    uint64 timestamp = 0;
    Status status = Active;
    string expirationDatetime = "";
    uint64 expirationTimestamp = 0;
    string reference = "839db975-3eb2-4303-ab9c-c208405d79a1";
}
```

**Response**

```
message OrderResponse {
    bool success = true;
    string reference = "839db975-3eb2-4303-ab9c-c208405d79a1";
}
```

Submitted orders typically go via consensus so the `OrderResponse` will only indicate that the message was accepted and sent out onto the blockchain to be included in a block. It could be rejected at a later stage of processing.

### Amend order

```
rpc AmendOrder(vega.Amendment) returns (OrderResponse);
```

To amend an existing order on the network, a caller can submit a protobuf `Amendment` message and receive an `OrderResponse` from the API. In the following example a trader wishes to amend an existing order with ID `v10028123-99091233` with a total of `1000` contracts at price `400` on market ID `BTC/DEC19`:

**Request**

```
message Amendment {
    string id = "v10028123-99091233";
    string party = "goldman";
    uint64 price = "400";
    uint64 size = "1000";
    string expirationDatetime = "";
    uint64 expirationTimestamp = 0;
}
```

**Response**

```
message OrderResponse {
    bool success = true;
    string reference = "839db975-3eb2-4303-ab9c-c208405d79a1";
}
```

Amendments typically go via consensus so the `OrderResponse` will only indicate that the message was accepted and sent out onto the blockchain to be included in a block. It could be rejected at a later stage of processing.


### Cancel order

```
rpc CancelOrder(vega.Order) returns (OrderResponse);
```

To cancel an existing order, a trader can submit a protobuf `order` message and receive an `OrderResponse` from the API. In the following example a trader wishes to `cancel` an existing active `order` with ID `v1008973-9376433` on market ID `BTC/DEC19`:

**Request**

```
message Order {
	string id = "v1008973-9376433"
    string market = "BTC/DEC19";
    string party = "goldman";
    Side side = Buy;
    uint64 price = 100;
    uint64 size = 500;
    uint64 remaining = 500;
    Type type = GTC;
    uint64 timestamp = 0;
    Status status = Active;
    string expirationDatetime = "";
    uint64 expirationTimestamp = 0;
    string reference = "839db975-3eb2-4303-ab9c-c208405d79a1";
}
```

**Response**

```
message OrderResponse {
    bool success = true;
    string reference = "839db975-3eb2-4303-ab9c-c208405d79a1";
}
```

Cancellations typically go via consensus so the `OrderResponse` will only indicate that the message was accepted and sent out onto the blockchain to be included in a block. It could be rejected at a later stage of processing.

## Benchmarks

There are two ways to run benchmarks:
* by using `go test -bench`; or
* by running `vegabench`.

To run benchmarks using `go test -bench`, run:
```bash
export GOMAXPROCS=4 # default 8
go test -run=XXX -bench=. -benchmem -benchtime=1s ./cmd/vegabench
# or simply: make bench
```
The output should look something like this:
```
BenchmarkMatching100-4  100  22525798 ns/op  13168713 B/op  24021 allocs/op
```
Output components:
* `BenchmarkMatching100` - the name of the test
* `-4` - the max number of processes (`$GOMAXPROCS`)
* _number_ - the number of times `go` ran the test so that it took longer than `benchtime`
* _number_ `ns/op` - average number of nanoseconds per operation
* _number_ `B/op` - average number of bytes allocated per operation
* _number_ `allocs/op` - average number of allocations per operation

To run benchmarks using `vegabench`, run:
```bash
make build # generate the vegabench binary
./cmd/vegabench/vegabench -orders 25000 -reportDuration 1s
```
For help with command arguments, run `vegabench --help`.

The output should look something like this:
```
(25.93%) Elapsed = 1s, average = 154.273µs
(54.10%) Elapsed = 2s, average = 147.961µs
(82.48%) Elapsed = 3s, average = 145.549µs
(n=25000) Elapsed = 4s, average = 143.386µs
```
Output components:
* _percentage_ - how far through the test we are
* `n=` _number_ - the total number of orders
* `Elapsed =` _time duration_ - how much time has passed since the start of the test; One output line should be printed once every `reportDuration`
* `average =` _time duration_ - running average (from beginning to now) time taken to match an order

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the vega and tendermint logs which are available on the console if running locally or by `multilog` if running on test networks. Default testnet location for log files:

* `/home/vega/log/vega/`
* `/home/vega/log/tendermint/`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.

Debugging the application locally is also possible with [Delve](https://github.com/go-delve/delve).
