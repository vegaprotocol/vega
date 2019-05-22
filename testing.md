We use `mockgen`, from [golang/mock](https://github.com/golang/mock).

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

The file `internal/candles/service.go` has:

```go
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/candles CandleStore
type CandleStore interface { /* ... */ }
```

In order to recreate just the candle mocks:

```bash
cd .../go/src/vega/internal/candles # trading-core
rm -rf mocks
go generate .
git diff # hopefully no differences
```

## Reasons for moving from mockery to mockgen

TBD (#230)
