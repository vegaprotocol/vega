# Getting Started

This document is a guide for new Go developers.

It starts with a vanilla Linux or MacOSX installation, and runs through all the
steps needed to build and test vega.

## Installing Golang

Almost all of vega (including tools not in this repo) are written in Go, so you will need it installed locally. The version targeted can be found in the `go.mod` at the root of this repo, but realistically there is not much harm in having a slightly newer version.

The Go tool-chain can be installed via an OS package manager, or directly from from https://golang.org/dl/. Use whichever you are most comfortable with. See also the [Golang installation guide](https://golang.org/doc/install) for more information.

After installation set the following environment variables:

```bash
# Add to $HOME/.bashrc:
export GOROOT="/path/to/your/go/install"
export GOPATH="$HOME/go"
export PATH="$PATH:$GOROOT/bin"
```

Now run the following to ensure everything exists and is in working order:

```bash
$ which go gofmt
/path/to/your/go/install/bin/go
/path/to/your/go/install/bin/gofmt

$ go version
go version go[INSTALLED VERSION] linux/amd64
```
## GitHub Authentication and Git configurations

To be able to clone/push/pull from github in a seamless way, it is worth setting up SSH keys in github so that authentication can happen magically. If not already set up, following this guide (https://github.com/settings/keys)

You also now need to tell git to prefer SSH over HTTPS when accessing all `vegaprotocol` repositories by doing the following:

```bash
git config --global url."git@github.com:vegaprotocol".insteadOf "https://github.com/vegaprotocol"
```

This is necessary since some of the repos that `vega` depends on in `vegaprotocol` are private repositories. The git setting ensure that `go get` now knows to use `ssh` too.


## MacOS Requirements

In order to get the required tools for MacOS make sure to install the following packages:
### `bash`
```bash
$ brew install bash
# now make sure you are using bash, not zsh (this can be tricky to modify)
```

### `jq`
```bash
$ brew install jq
```

### `gnu-sed`
```bash
$ brew install gnu-sed
# read the stdout, cos it asks you to modify `.profile` or `.bash_profile`
# e.g. add export PATH="/usr/local/Cellar/gnu-sed/4.8/libexec/gnubin:$PATH"
```

### `coreutils`
```bash
$ brew install coreutils
# again read the stdout - similar changes required to modify `.profile` or `.bash_profile`
# e.g export PATH="/usr/local/Cellar/coreutils/9.0/libexec/gnubin:$PATH" 
```

### `findutils`
```bash
$ brew install findutils
# again read the stdout to modify `.profile` or `.bash_profile`
# e.g. export PATH="/usr/local/Cellar/findutils/4.8.0_1/libexec/gnubin:$PATH"
```

## Building and Testing Vega

Go makes building easy:

```bash
git clone git@github.com:vegaprotocol/vega.git
cd vega

go install ./...

# check binary works
vega --help
```

And equally also makes testing easy:

```bash
go test ./...
go test -race ./...
go test -v ./integration/... --godog.format=pretty
```

There is also a `Makefile` which contain the above commands and also some other useful things.

## Running A Vega Node Locally

With vega built it is technically possible to run the node locally, but it is a bit cumbersome. The steps are here if you are feeling brave: https://github.com/vegaprotocol/networks

An alternative is to use `VegaCapsule` (VC) which will allow you to confuigure and run a network locally. For more information and  detailed information to get started see the [VC repo](https://github.com/vegaprotocol/vegacapsule)
