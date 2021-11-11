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

An alternative is to use `dockerizedvega` (DV) which will trivially spin up a working system for you. The script and some detailed information can be found here: https://github.com/vegaprotocol/devops-infra/blob/master/scripts/dockerisedvega.sh

In summary you just need to do the following (Note that if you are on MacOS and probably also Windows you may need to increase the allocated memory to 4GB using the Docker Desktop UI):

```
dockerisedvega.sh --vega-loglevel DEBUG --prefix mydvbits --portbase 1000 --validators 2 --nonvalidators 1 start

dockerisedvega.sh --vega-loglevel DEBUG --prefix mydvbits --portbase 1000 --validators 2 --nonvalidators 1 stop
```

This will pull images containing the latest versions of all the vega tools. To inject a locally built vega into DV you need to build a new image. This can be done using the following script:

```
#!/bin/bash

# If on a Mac we will need to cross-compile
export GOOS=linux
export GOARCH=amd64

go build -v -gcflags "all=-N -l" -o "cmd/vega/vega-dbg-lin64" "./cmd/vega"

mkdir -p docker/bin
cp -a cmd/vega/vega-dbg-lin64 docker/bin/vega

# remove any existing image with that tag
docker rmi docker.pkg.github.com/vegaprotocol/vega/vega:local -f

docker build -t "docker.pkg.github.com/vegaprotocol/vega/vega:local
```

with this you can then run the DV start line again with the addition of the option `--vega-version local`.

## Other Things to Try and Build

There are other repos that you will probably need to touch at some point so it is worth trying to build those too. Having completed the above you will be in a good place to do this. Have a fiddle in these repos:
- `vegawallet`
- `data-node`
- `vegatools`
- `protos` (will involve some more go getting)




