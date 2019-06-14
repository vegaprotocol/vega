This document is a guide for new Go developers which starts with a vanilla Linux
or MacOSX installation, and runs through all the steps needed to get a working
single-node Vega system.

A Vega system backend requires Vega (from the `trading-core` repo) and
`tendermint` (third party open source software providing Byzantine Fault
Tolerant (BFT) state machine replication, ie Blockchain).

## System packages

The following OS packages are required:

* `make`

## Installing golang

**Required version: 1.11.5**

Get Golang via OS package manager, or directly from from https://golang.org/dl/.
Install it somewhere, then point "`$GOROOT`" at that location:

```bash
# Add to $HOME/.bashrc:
export GOROOT="/path/to/your/go1.11.5"
export PATH="$PATH:$GOROOT/bin"
```

Ensure you have `go`, `godoc` and `gofmt`:

```bash
$ which go godoc gofmt
/path/to/your/go1.11.5/bin/go
/path/to/your/go1.11.5/bin/godoc
/path/to/your/go1.11.5/bin/gofmt

$ go version
go version go1.11.5 linux/amd64
```

## Set up Go source paths and Go Modules

At present (June 2019), Go Modules are in the process of being introduced to the
Go ecosystem, so things are a little clunky. There are several ways of getting
things working. The main options are:

* Set `GO111MODULE` to `auto`. Install source that **uses** Go Modules
  **outside** `$GOPATH` and source that **does not use** Go Modules **inside**
  `$GOPATH`.
* Set `GO111MODULE` to `on`. Install all source **inside** `$GOPATH`. Remember
  that source that does not use Go Modules will have to be treated differently.

This document works with the second option (`GO111MODULE=on`).

All Vega Golang repositories have been set up to use Go Modules (check for files
`go.mod` and `go.sum` in the top-level directory).

Create directories `$HOME/go/bin`, `$HOME/go/pkg`, and `$HOME/go/src`, then
point `$GOPATH` at this location:

```bash
# Add to $HOME/.bashrc
export GOPATH="$HOME/go"
export PATH="$PATH:$HOME/go/bin"
export GO111MODULE=on # or auto
```

## Gitlab Auth

Either use your existing Gitlab account, or create a Vega-specific one.

If not already present (in `$HOME/.ssh`), create an RSA keypair:

```bash
ssh-keygen -t rsa -b 4096
```

Add the public key (found in `$HOME/.ssh/id_rsa.pub`) to Gitlab:
https://gitlab.com/profile/keys

## Get trading-core

```bash
cd $GOPATH/src
git clone git@gitlab.com:vega-protocol/trading-core.git vega
cd vega
git status # On branch develop, Your branch is up to date with 'origin/develop'.

make gettools # get the build tools

make deps # get the source dependencies
make gqlgen_check # warning: This may take a minute, with no output.
make proto_check
make install

# Now check:
git rev-parse HEAD | cut -b1-8
vega --version
# hashes should match.
```

## Get tendermint

TBD

## Running vega

TBD

## Running tendermint

TBD

## Running go-trade-bot

TBD
