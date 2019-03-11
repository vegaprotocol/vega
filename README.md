# Vega

A decentralised trading platform that allows pseudo-anonymous trading of derivatives on a blockchain.

**Vega** provides the following core features:

<img align="right" src="https://vegaprotocol.io/img/concept_header.svg" height="170" style="padding: 0 10px 0 0;">

- Join a Vega network as a validator or non-concensus node. 
- [Provision](#provisioning) new markets on a network (coming soon)
- [Manage orders](#orders) (and [trade on a network](#trading))
- [Configure a node](#configuration) (and it's [APIs](#apis))
- Manage [authentication](#auth) with a network (coming soon) 
- [Run scenario tests](#scenario-tests) (coming soon)
- [Run benchmarks](#benchmarks) and test suites (coming soon)
- Support [settlement](#settlement) in cryptocurrency (coming soon) 

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
- [Scenario tests](#scenario-tests)
- [Benchmarks](#benchmarks)
- [Scripts](#scripts)
- [Releasing](#releasing)
- [Troubleshooting & debugging](#troubleshooting--debugging)
- [Resources](#resources)
- [Credits](#credits)

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

### Local (Coming soon)

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

Vega is initialised with a set of default configuration with the command `vega init`. There are [plenty of options](/config.toml) to configure it. To override any of the defaults edit your `config.toml` typically found in the `~/.vega` directory:

## APIs

(coming soon)

## Provisioning

The provisioning of new markets is **coming soon**. 

Vega supports a single fixed market with ID `BTC/DEC19` which can be passed to APIs as the field `Market` in protobuf/rest/graphql requests.

## Trading

(coming soon)

## Scenario tests

(coming soon)

## Benchmarks

(coming soon)

## Scripts

(coming soon)

## Releasing

(coming soon)

## Troubleshooting & debugging

(coming soon)

## Resources

(coming soon)

## Credits

(coming soon)
