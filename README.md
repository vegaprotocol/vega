## Vega Core

[![coverage report](https://gitlab.com/vega-protocol/trading-core/badges/develop/coverage.svg)](https://gitlab.com/vega-protocol/trading-core/commits/develop)

The core trading application logic, Tendermint blockchain, and communications modules.

### Building it

First, install the Dep dependency manager, see https://golang.github.io/dep/

Then, assuming a default `$GOPATH`, clone the source code into `~/go/src/vega`. If your `$GOPATH` is somewhere else, that's fine too.

```
mkdir -p ~/go/src/vega
git clone git@gitlab.com:vega-protocol/trading-core.git ~/go/src/vega
```

Once you've got it, do a `dep ensure`. The proper version of each dependency will be downloaded.

`build.sh` will create an executable called `vega` which you can run. Alternately, `go run cmd/vega/main.go` will run the checked-out code, just as you'd expect.

### Installing and Running Tendermint

We're using Tendermint for distributing transactions across multiple nodes.

Install docs are here: http://tendermint.readthedocs.io/projects/tools/en/master/install.html

Once you've built Tendermint, start Vega like this:

```
# initialize tendermint
tendermint init

# run vega with the chain and http server
vega --chain

# start creating blocks with Tendermint
tendermint node
```

At this point, you've got:

* a Vega blockchain TCP socket open on port 46658, connected to Tendermint
* a stub REST API running on [http://localhost:3001](http://localhost:3001)
* a stub Server Sent Events (SSE) API pushing events out at http://localhost:3002/events/orders

Tips:

* if you need to reset the chain, run `tendermint unsafe_reset_all`

### Adding new dependencies

Do a `dep ensure -add github.com/foo/bar` to add to the manifest.

### Deploying

Deployments are automated using Capistrano. Currently the `staging` environment points at Dave's `x.constructiveproof.com` servers. A few commands to note:

* `cap staging vega:full_reset` will build the `vega` binary locally, stop tendermint and vega, upload the binary, blow away all previous chain data, and restart vega and tendermint on all staging servers.

* `cap staging:reset_app_servers` resets everything but does not build and upload the latest binary.

TODO: A better deploy process wouldn't be tied to Dave's account on those servers. This is currently in progress.

### Documentation

* API Documentation is expressed in [Swagger YAML 2.0](https://swagger.io/docs/specification/2-0/basic-structure/) format, hosted in the [trading-api-docs repo](https://gitlab.com/vega-protocol/trading-api-docs).
* It can be used to generate models and REST endpoints using [Go-Swagger](https://github.com/go-swagger/go-swagger) (which itself has some [documenation](https://goswagger.io/).
* Note that currently validation isn't working for some reason.

#### Workflow

1) Discuss some changes with your team and update [swagger.yaml](https://gitlab.com/vega-protocol/trading-api-docs/blob/master/swagger.yaml) in the [trading-api-docs repo](https://gitlab.com/vega-protocol/trading-api-docs).

2) Regenerate stuff

```
sh code_gen.sh
```

3) Serve docs locally

```
// ReDoc
swagger serve

// Swagger
swagger server -F swagger
```