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

**Required version: 1.11.13**

Get Golang via OS package manager, or directly from from https://golang.org/dl/.
See also the [Golang installation guide](https://golang.org/doc/install).
Install it somewhere, then point "`$GOROOT`" at that location:

```bash
# Add to $HOME/.bashrc:
export GOROOT="/path/to/your/go1.11.13"
export PATH="$PATH:$GOROOT/bin"
```

Ensure you have `go`, `godoc` and `gofmt`:

```bash
$ which go godoc gofmt
/path/to/your/go1.11.13/bin/go
/path/to/your/go1.11.13/bin/godoc
/path/to/your/go1.11.13/bin/gofmt

$ go version
go version go1.11.13 linux/amd64
```

## Setup Go source path

We use go-mod, so we will be checking the code out, outside the go path. 

* e.g. git clone git@gitlab.com:vega-protocol/trading-core.git ~/Code/Vega

All Vega Golang repositories have been set up to use Go Modules (check for files
`go.mod` and `go.sum` in the top-level directory).

## GitLab Authentication

Either use your existing GitLab account, or create a Vega-specific one.

If not already present (in `$HOME/.ssh`), create an RSA keypair:

```bash
ssh-keygen -t rsa -b 4096
```

Add the public key (found in `$HOME/.ssh/id_rsa.pub`) to GitLab:
https://gitlab.com/profile/keys

## Get trading-core

The `trading-core` repo uses Vega's `quant` repo. Ensure Go knows to use `ssh`
instead of `https` when accessing repositories stored at GitLab:

```bash
git config --global url."git@gitlab.com:".insteadOf "https://gitlab.com/"
```

Next, clone `trading-core`:

```bash
cd $GOPATH/src
git clone git@gitlab.com:vega-protocol/trading-core.git vega
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

**Required version: 0.31.5**

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
* To remove Vega store content then run a Vega node, use:

  ```bash
  rm -rf "$HOME/.vega/"*store
  vega node
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
* Optional: To run a multi-node network, use `tendermint testnet` to generate
  configuration files for nodes.

## Developing trading-core

In order to develop trading core, more tools are needed. Install them with:

```bash
# get the dev tools
make gqlgen_check # warning: This may take a minute, with no output.
make proto_check
```

## Running Traderbot

Clone Traderbot from https://gitlab.com/vega-protocol/traderbot/ into
`$GOPATH/src`.

Build: `make install`

Run: `traderbot -config configfiles/localhost.yaml`

Start traders: `curl --silent -XPUT "http://localhost:8081/traders?action=start"`
