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

We're using Tendermint (please use v0.26.0) for distributing transactions across multiple nodes.

Install docs are here: http://tendermint.readthedocs.io/projects/tools/en/master/install.html

We recommend downloding a pre-built binary for your architecture rather than compiling from source.

Once you've built Tendermint, we need to create several directories for persisting data (in future releases this will be configurable):

```
cd <vega_binary_dir>
mkdir ./tmp
mkdir ./tmp/orderstore
mkdir ./tmp/candlestore
mkdir ./tmp/tradestore
```

Next we initialise Tendermint and update config:

```
# initialize tendermint
tendermint init

# edit config.toml files and replace all references to port 266** to 466**
nano ~/.tendermint/config/config.toml
```

Finally, we start Vega like this:

```
# initialize tendermint
tendermint init

# run vega with the chain and http server
vega

# start creating blocks with Tendermint
tendermint node
```

At this point, you've got:

* a Vega blockchain TCP socket open on port 46658, connected to Tendermint
* a stub GraphQL API running on [http://localhost:3004/](http://localhost:3004) including 'Playground' to test out queries/mutations/subscriptions
* a stub REST API running on [http://localhost:3003](http://localhost:3003)
* a stub gRPC API running on [http://localhost:3002/](http://localhost:3002)

Tips:

* if you need to reset the chain, run `tendermint unsafe_reset_all`

### Adding new dependencies

Do a `dep ensure -add github.com/foo/bar` to add to the manifest.

### Deploying to dev-net & test-net

Generally speaking, testing can be done against a local Tendermint and Vega binary, and deployments to our 'live' nets should be performed carefully. For example: Test-net needs to be up and running for investor demos. 

Deployments are automated using Capistrano. ***Important:*** state management of dev-net and test-net should be performed using the capistrano scripts in the ***reset-service*** repo and not from the trading-core.

* `cap devnet vega:build` will build the `vega` binary locally.
* `cap devnet vega:upload` will upload the `vega` binary to the remote server specified in capistrano config.

TODO: A better deploy process not including capistrano, driven from CI