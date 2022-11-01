# Data node

Version 0.60.0

A service exposing read only APIs built on top of [Vega](https://github.com/vegaprotocol/vega) platform.

**Data node** provides the following core features:

- Consume all events from Vega core
- Aggregates received events and stores the aggregated data
- Serves stored data via [APIs](#apis)
- Allows advanced configuration [Configure a node](#configuration)

## Links

- For **new developers**, see [Getting Started](../GETTING_STARTED.md).
- For **updates**, see the [Change log](../CHANGELOG.md) for major updates.
- For **architecture**, please read the [documentation](docs/index.md) to learn about the design for the system and its architecture.
- Please [open an issue](https://github.com/vegaprotocol/data-node/issues/new) if anything is missing or unclear in this documentation.

<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Data node](#data-node)
  - [Links](#links)
  - [Installation and configuration](#installation-and-configuration)
  - [Troubleshooting & debugging](#troubleshooting--debugging)

<!-- tocstop -->

</details>

## Installation and configuration

To install see [Getting Started](https://docs.vega.xyz/mainnet/node-operators/setup-datanode).

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the Vega and Tendermint logs which are available on the console if running locally or by journal plus syslog if running on test networks. Default location for log files:

* `/var/log/vega.log`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.
