# Coding style

This document serves to outline the coding style standard we aim to follow throughout the codebase.

## Starting point

As a starting point, we should follow the official golang `CodeReviewComments` document [found on github](https://github.com/golang/go/wiki/CodeReviewComments). The basics are:

* names are `camelCased` or `CamelCased`, for functions, types, and constants alike.
* Avoid stutter (`markets.NewMarket` should be `markets.New()` etc...)
* Code should be passed though the `gofmt` tool
* ...

## Superset

As a basis, we're using the golang `CodeReviewComments`. We're adding a few things to that that either have proven to be issues in our codebase, or things that are considered _good practice_, and we therefore want to enforce.

1. Use directional channels whenever possible. The gRPC streams are used to subscribe to updates to specific bits of data. Internally, these subscriptions take the shape of channels. Because streams are read-only, the channels created and passed around are, by definition read-only. The return types should reflect this:

```go
// SubscribeToSomething the simplest example of a subscription
func (t *T) SubscribeToSomething(ctx context.Context) (<-chan *types.Something, error) {
    if err := t.addSubscription(); err != nil {
        return nil, err
    }
    ch := make(chan *types.Something)
    go func () {
        defer func() {
            t.rmSubscription()
            close(ch)
        }()
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            default:
                for _, s := range t.latestSomethings() {
                    ch <- s
                }
            }
        }
    }()
    return ch, nil
}
```

2. Unit tests cover the exported API. A single exported function performs a straightforward task (input X produces output Y). How that logic is implemented is not relevant. Whether the functionality is implemented in the function body, or spread out over 20 unexported functions is irrelevant. The implementation details should be something we can refactor, and we should be able to verify the validity of the refactor by running the tests and still have them pass. To this end, unit tests are defined in a `package_test` package.
All dependencies of the tested unit are mocked using `mockgen`, and expected calls to dependencies are checked using the mocks.

3. The interface any given package uses is defined inside the package itself, not in the package that implements the required interface.

4. Constructor functions (`New`) return types, but accept interfaces. For example, an engine constructor may depend on a buffer, the constructor should look like this:

```go
// FooBuffer ..

//go:generate go run github.com/golang/mock/mockgen -destination mocks/foo_buffer_mock.go -package mocks code.vegaprotocol.io/vega/foo FooBuffer
type FooBuffer interface {
    Add(types.Foo)
}

// NewFooEngine returns new foo engine, requires the foo buffer
func NewFooEngine(fooBuf FooBuffer) *fooEngine {
    return &fooEngine{
        buf: fooBuf,
    }
}
```

## Protobuf

In addition to the golang code review standards, we want to be consistent:

* Avoid nested types as much as possible. Enums are the notable exception here.
* Fields that are ID's should be named `ID` or `FooID` (ID is CAPS).
* Messages used in the API use the suffix `Request` and `Response`.
* API Request/Response types, and the service definitions belong in the `proto/api` directory (and the `api` package)
* Message types representing a unit of data, currently used in the core (e.g. `Order`, `Market`, `Transfer`, ...) are defined in the `proto/vega.proto` file. These types are imported under the alias `types`.
* Wherever possible, add validator tags to the proto definitions.

### Example

```proto
message Meta {
    string key = 1;
    string value = 2;
}

message Something {
    enum Status {
        Disabled = 0;
        Enabled = 1;
    }
    string ID = 1 [(validator.field) = {string_not_empty : true }];
    string marketID = 2;
    string partyID = 3;
    Status status = 4;
    repeated Meta meta = 5;
}
```

This generates the following types:

```go
type Something struct {
    ID        string
    MarketID  string
    PartyID   string
    Status    Status
    Meta      []Meta
}

type Meta struct {
    Key    string
    Value  string
}

type Something_Status int32

const (
    Something_Disbaled Something_Status = 0
    Something_Enabled  Something_Status = 1
)
```

To add an RPC call to get this _"something"_ from the system, add a call to the trading_data service in proto/api/trading.proto:

```proto
service trading_data {
    rpc GetSomethingsByMarketID(GetSomethingsByMarketIDRequest) returns (GetSomethingsByMarketIDResponse);
}

message GetSomethingsByMarketIDRequest {
    string marketID = 1 [(validator.field) = {string_not_empty : true}];
}

message GetSomethingsByMarketIDResponse {
    repeated vega.Something something = 1;
}
```
