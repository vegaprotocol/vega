# Data node

Version 0.53.0

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

## PostgreSQL
As of version 0.53, data node uses [PostgreSQL](https://www.postgresql.org) as its storage back end instead of the previous mix of in-memory and BadgerDB file stores. We also make use of Postgres extension called [TimescaleDB](https://www.timescale.com), which adds a number of time series specific features.

Postgres is not an embedded database, but a separate server application that needs to be running before datanode starts, and a side effect of this transition is a little bit of setup is required by the data node operator.

By default, data node will attempt to connect to a database called `vega` listening on `localhost:5432`, using the username and password `vega`. This is of course all configurable in data node’s `config.toml` file.

We are developing using `PostgreSQL 14.2` and `Timescale 2.7.1` and _strongly recommend_ that you also use the same versions.

```json
​​[SQLStore]
 UseEmbedded = false
 [SQLStore.ConnectionConfig]
   Host = "localhost"
   Port = 5432
   Username = "vega"
   Password = "vega"
   Database = "vega"
   UseTransactions = true

```
### Persistence
Currently the database is destroyed if it exists and recreated at data node start-up, though we expect this to change in the not too distant future once the schema has settled down and we add support for starting/stopping data nodes without replaying the entire chain.

There are a few different ways you can get postgres & timescale up and running.

### Using docker
This is probably the most straightforward and reliable way to get up and running.

Timescale supply a docker image, so assuming you [already have](https://www.docker.com/get-started/) docker installed, it is a simple matter of:

```sh
docker run --rm \
           -d
           -e POSTGRES_USER=vega \
           -e POSTGRES_PASSWORD=vega \
           -e POSTGRES_DB=vega \
           -p 5432:5432 \
           timescale/timescaledb:2.7.1-pg14
```

### Using your operating system's native packages

Timescale [have a set of instructions](https://docs.timescale.com/install/latest/self-hosted/) for installing Postgres/Timescale using `.deb` or `.rpm` they have built. If you follow these and get postgres running as a system service you'll then have to create a database, user, and password for the data node to use. For example:

```sql
➜  ~ sudo -u postgres psql
psql (14.3 (Ubuntu 14.3-0ubuntu0.22.04.1))
Type "help" for help.


postgres=# create database vega;
CREATE DATABASE

postgres=# create user vega with password 'vega';
CREATE ROLE

postgres=# grant all privileges on database vega to vega;
GRANT
```

### Using 'embedded' PostgreSQL
As mentioned above, PostgreSQL is not an embedded database. However, the good folks over at [embedded-postgres-go](https://github.com/fergusstrange/embedded-postgres) didn't let that stop them trying.

This go package allows us to start a PostgreSQL server from the data-node. It does this by
- Examining your system to figure out what platform/architecture it is
- Downloading an appropriate PostgreSQL binary installation
- Unpacking it to a temporary location
- Configuring and launching Postgres as a child process of data-node

embedded-postgres-go doesn't come with support for TimescaleDB so we forked it and built a set of our own binaries for a limited set of platforms which we [host on GitHub](https://github.com/vegaprotocol/embedded-postgres-binaries/releases/).

We use it for running integration tests and it works quite well however, we haven't tested it on a wide range of platforms, and ran into a few odd issues usually related to linking to various system libraries or sometimes not shutting down cleanly.

You can launch postgres in this way either with the command either using

```sh
data-node postgres run
```

Which will launch embedded postgres in it's own process or

Or by setting
```json
​​[SQLStore]
  UseEmbedded = true
```

Which will cause data-node to launch Postgres as it starts up, and stop it when it exits. While convenient, if data-node is forcefully killed and doesn't have chance to shutdown it is possible for postgres to keep on running. Postgres then needs to be manually killed to prevent 'unable to bind to port' errors on the next start.

In both cases, the files for the database will be stored in your 'state' directory, e.g. `~/.local/state/vega/data-node/` on Linux.

### Building from source

It's quite straightforward; if this is your preferred option you probably already know how to do it. There are instructions on the timescale website.

### Using a cloud database provider

This isn't something we've tested yet, but it's something we plan to investigate in the future. Feel very free to give it a try; our main concern is that the latency of the connection may cause data-node to be unable to process blocks as fast as they are produced.

Timescale provide a hosted service, I believe `AWS` do as well.
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

#### GraphQL SSL

**GraphQL subscriptions do not work properly unless the HTTPS is enabled**.

To enable TLS on the GraphQL port, set
```toml
  [Gateway.GraphQL]
    HTTPSEnabled = true
```

You will need your data node to be reachable over the internet with a proper fully qualified domain name, and a matching certificate. If you already have a certificate and corresponding private key file, you can specify them as follows:
```toml
  [Gateway.GraphQL]
    CertificateFile = "/path/to/certificate/file"
    KeyFile = "/path/to/key/file"
```

If you prefer, the data node can manage this for you by automatically generating a certificate and using `LetsEncrypt` to sign it for you.

```toml
  [Gateway.GraphQL]
    HTTPSEnabled = true
    AutoCertDomain = "my.lovely.domain.com"
```

However, it is a requirement of the `LetsEncrypt` validation process that the the server answering its challenge is running on the standard HTTPS port (443). This means you must either
- Forward port 443 on your machine to the GraphQL port (3008 by default) using `iptables` or similar
- Directly use port 443 for the GraphQL server in data-node by specifying
```toml
  [Gateway.GraphQL]
    Port = 443
```
Note that Linux systems generally require processes listening on ports under 1024 to either
  - run as root, or
  - be specifically granted permission, e.g. by launching with
  ```
  setcap cap_net_bind_service=ep data-node run
  ```

### REST

REST provides a standard between computer systems on the web, making it easier for systems to communicate with each other. It is arguably simpler to work with than gRPC and GraphQL. In Vega the REST API is a reverse proxy to the gRPC API, however it does not support streaming.

The default port (configurable) for the REST API is `3009` and we use a reverse proxy to the gRPC API to deliver the REST API implementation.

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the Vega and Tendermint logs which are available on the console if running locally or by journal plus syslog if running on test networks. Default location for log files:

* `/var/log/vega.log`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.
