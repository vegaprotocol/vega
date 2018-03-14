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

Tips:

* if you need to reset the chain, run `tendermint reset_unsafe_all`
