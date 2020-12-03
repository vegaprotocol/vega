# Getting Started

This document is a guide for new Go developers.

It starts with a vanilla Linux or MacOSX installation, and runs through all the
steps needed to get a working single-node Vega system.

A Vega system back end requires Vega (from the `trading-core` repo) and
`tendermint` (third party open source software providing Byzantine Fault
Tolerant (BFT) state machine replication, i.e. blockchain).

## System packages

The following OS packages are required:

* `bash` (or a shell of your choice, but this document assumes `bash`)
* `make`

## Installing Golang

**Required version: 1.14.4**

Get Golang via OS package manager, or directly from from https://golang.org/dl/.
See also the [Golang installation guide](https://golang.org/doc/install).
Install it somewhere, then point "`$GOROOT`" at that location:

```bash
# Add to $HOME/.bashrc:
export GOROOT="/path/to/your/go1.14.4"
export PATH="$PATH:$GOROOT/bin"
```

Ensure you have `go` and `gofmt`:

```bash
$ which go gofmt
/path/to/your/go1.14.4/bin/go
/path/to/your/go1.14.4/bin/gofmt

$ go version
go version go1.14.4 linux/amd64
```

## Set up Go source path

At present (June 2019), Go Modules are in the process of being introduced to the
Go ecosystem, so things are a little clunky. There are several ways of getting
things working. The default option used in this project is:

* Set `GO111MODULE` to `on`. Install all source **inside** `$GOPATH`.
  Remember that source that does not use Go Modules will have to be treated
  differently.

For advanced Golang users who are happy to support the system themselves:

* Set `GO111MODULE` to `auto`. Install source that **uses** Go Modules
  **outside** `$GOPATH` and source that **does not use** Go Modules **inside**
  `$GOPATH`.

All Vega Golang repositories have been set up to use Go Modules (check for files
`go.mod` and `go.sum` in the top-level directory).

## GitHub Authentication

Either use your existing GitHub account, or create a Vega-specific one.

If not already present (in `$HOME/.ssh`), create an RSA keypair:

```bash
ssh-keygen -t rsa -b 4096
```

Add the public key (found in `$HOME/.ssh/id_rsa.pub`) to GitHub:
https://github.com/settings/keys

## Get vega

The `vega` repo uses Vega's `quant` repo. Ensure Go knows to use `ssh`
instead of `https` when accessing `vegaprotocol` repositories on Github:

```bash
git config --global url."git@github.com:vegaprotocol".insteadOf "https://github.com/vegaprotocol"
```

Next, clone `vega`:

```bash
cd $GOPATH/src
git clone git@github.com:vegaprotocol/vega.git
cd vega
git status # On branch develop, Your branch is up to date with 'origin/develop'.

make gettools_build # get the build tools
make deps # get the source dependencies
make install # build the binaries and put them in $GOPATH/bin

# Now check:
git rev-parse HEAD | cut -b1-8
vega --version
# hashes should match.
```

## Get Tendermint

**Required version: 0.33.8**

[Tendermint](https://tendermint.com/docs/introduction/what-is-tendermint.html)
performs Byzantine Fault Tolerant (BFT) state machine replication (SMR) for
arbitrary deterministic, finite state machines. It is required for Vega nodes to
communicate.

It is quicker and easier to download a pre-built binary, rather than compiling
from source.

Download Tendermint from https://github.com/tendermint/tendermint/releases/.
Install the binary somewhere on `$PATH`. If needed, see also the
[Tendermint installation guide](https://tendermint.com/docs/introduction/install.html).

## Running Vega

* To create a new configuration file, use:

  ```bash
  vega init -f
  ```
this will trigger a password prompt which will be used to encrypt your vega nodewallet.

If used in automation you can specify a file containing the password:
```bash
vega init -f --nodewallet-password="path/to/file"
```

you can also generate dev usage wallet for all vega supported foreign chains:
```bash
vega init -f --gen-dev-nodewallet
```

* To remove Vega store content then run a Vega node, use:

  ```bash
  rm -rf "$HOME/.vega/"*[r,e]store
  vega node
```

If used in automation you can specify a file containing the password:
```bash
vega node --nodewallet-password="path/to/file"
```

## Running Tendermint

* The version must match the required version (above). To check the version,
  use:
  ```bash
  tendermint version
  ```

* To create a new configuration file, use:
  ```bash
  tendermint init
  ```
* To remove chain data (go back to genesis) then run a Tendermint node, use:

  ```bash
  tendermint unsafe_reset_all
  tendermint node
  ```
* At this stage, you should be able to watch the block production (an other Tendermint events) using:
  ```bash
  vega watch "tm.event = 'NewBlock'"
  ```

* Optional: To run a multi-node network, use `tendermint testnet` to generate
  configuration files for nodes.

## Developing trading-core

In order to develop trading core, more tools are needed. Install them with:

```bash
# get the dev tools
make gettools_develop
make gqlgen_check # warning: This may take a minute, with no output.
make proto_check
```

## Running Traderbot

Clone Traderbot from https://github.com/vegaprotocol/traderbot/ into
`$GOPATH/src`.

Build: `make install`

Run: `traderbot -config configfiles/localhost.yaml`

Start traders: `curl --silent -XPUT "http://localhost:8081/traders?action=start"`
