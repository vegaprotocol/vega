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

**Required version: 1.16.2**

Get Golang via OS package manager, or directly from from https://golang.org/dl/.
See also the [Golang installation guide](https://golang.org/doc/install).
Install it somewhere, then point "`$GOROOT`" at that location:

```bash
# Add to $HOME/.bashrc:
export GOROOT="/path/to/your/go1.16.2"
export PATH="$PATH:$GOROOT/bin"
export GOPRIVATE=github.com/vegaprotocol,code.vegaprotocol.io
```

Ensure you have `go` and `gofmt`:

```bash
$ which go gofmt
/path/to/your/go1.16.2/bin/go
/path/to/your/go1.16.2/bin/gofmt

$ go version
go version go1.16.2 linux/amd64
```

## Set up Go source path

At present (June 2019), Go Modules are in the process of being introduced to the
Go ecosystem, so things are a little clunky. There are several ways of getting
things working. The default option used in this project is:

* Set `GO111MODULE` to `on`. Install all source **inside** `$GOPATH`.
  Remember that source that does not use Go Modules will have to be treated
  differently.

* Set `GONOSUMDB` to `code.vegaprotocol.io/*` to assure that private package dependencies can be installed. This is because data node relies on some private vega packages.

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

## Get data node

The `data-node` repo uses Vega's `protos` and `vega` repos. Ensure Go knows to use `ssh`
instead of `https` when accessing `vegaprotocol` repositories on Github:

```bash
git config --global url."git@github.com:vegaprotocol".insteadOf "https://github.com/vegaprotocol"
```

Next, clone `data-node`:

```bash
cd $GOPATH/src
git clone git@github.com:vegaprotocol/data-node.git
cd data-node
git status # On branch develop, Your branch is up to date with 'origin/develop'.
```

## Get Vega core node

The data node on it's own doesn't do much as it relies on Vega core node to send events to it.
Because of that it is vital to run Vega core node next to it.
Please refer to [Vega Getting Started](https://github.com/vegaprotocol/vega/blob/develop/GETTING_STARTED.md).

## Running data node

* Build node from source
```
go install ./cmd/data-node
```

* To initiate data node with a default configuration, use:

  ```bash
  data-node init -r ~/.data-node
  ```

By running this command data node will initiates basic folder structure for data storage and creates default configuration file.

Configuration file can be find in:

```bash
cat ~/.data-node/config.toml
```

* To remove data node store content then run a data node, use:

  ```bash
  rm -rf "~/.data-node"*[r,e]store
  data-node node -r ~/.data-node
```

## Developing data node

In order to develop data node, more tools are needed. Install them with:

```bash
# get the dev tools
make gettools_develop
make gqlgen_check # warning: This may take a minute, with no output.
make proto_check
```