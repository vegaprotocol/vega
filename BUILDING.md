# Building

## Setting up your environment
``` code
# This is where the executables will be built
export GOBIN=/home/someuser/bin
# This is the root folder of your golang installation
export GOROOT=/usr/local/go
# This tells the tool chain not to use C only libraries
export CGO_ENABLED=false
```

## Pulling down the source code
``` script
git clone https://github.com/vegaprotocol/vega
cd vega
```

If you need a specific version you can get that by running

``` script
git checkout vX.XX
``` 

## Install golang
The version of golang required can be found at the top of the `go.mod` file in the root of the vega repository. To install the software follow the link below and choose the correct version and platform you require. Then follow the instructions given.

`https://go.dev/dl/`


## Building the executables

``` script
go install ./...
```

This will download any required dependencies and then build the executables and move them to the folder you selected with the GOBIN variable above.

Check in the `$GOBIN` folder that you have at least the following executables

* `vega`
* `vegawallet`
* `data-node`
* `visor`





