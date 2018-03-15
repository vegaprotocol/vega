## Vega Core

The core trading application logic, Tendermint blockchain, and communications modules.

### Building it

First, install the Glide dependency manager, see: https://glide.sh/

Once you've got it, do a `glide install`. The proper version of each dependency will be downloaded into your local filesystem.

`go build` will create an executable called `vega` which you can run. Alternately, `go run main.go` will run the checked-out code, just as you'd expect.

### Installing and Running Tendermint

We're using Tendermint for distributing transactions across multiple nodes.

Install: `go get github.com/tendermint/tendermint/cmd/tendermint`

That will build the `tendermint` binary. Assuming your `$GOBIN` works, initialize it like this:

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

* if you need to reset the chain, run `tendermint reset_unsafe_all`

### Adding new dependencies

Do a `glide install github.com/foo/bar`. We'll be moving to the new `godep` dependency manager soon, but for the moment Glide's working fine.
