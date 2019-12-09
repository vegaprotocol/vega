We use `mockgen`, from [`golang/mock`](https://github.com/golang/mock).

## Generating/updating mock files

```bash
# Note: this may take over a minute, with no output.
make mocks
```

The Makefile target runs `go generate`, which looks for comments of the form
`//go:generate`.

Because these comments have `go run github.com/golang/mock/mockgen` as the
command, there is no need to run `go get` or `go install` to fetch or install
`mockgen`.

## Example

The file `candles/service.go` has:

```go
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/candles CandleStore
type CandleStore interface { /* ... */ }
```

In order to recreate just the candle mocks:

```bash
cd .../go/src/vega/candles # trading-core
rm -rf mocks
go generate .
git diff # hopefully no differences
```

## Running tests

To run all tests, use:

```bash
make test
```

To run tests from one subdirectory, use:

```bash
go test ./somedir/
```

To force a re-run of previously successful tests, add `-count 1`.

## Reasons for moving from `mockery` to `mockgen`

TBD (#230)
