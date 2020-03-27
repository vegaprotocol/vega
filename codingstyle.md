# Coding style

This document serves to outline the coding style standard we aim to follow throughout the codebase.

## Starting point

As a starting point, we should follow [the official golang `CodeReviewComments` document](https://github.com/golang/go/wiki/CodeReviewComments). The basics are:

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
    Something_Disabled Something_Status = 0
    Something_Enabled  Something_Status = 1
)
```

To add an RPC call to get this _"something"_ from the system, add a call to the `trading_data` service in `proto/api/trading.proto`:

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

### By popular demand:

Named return values are perfectly fine. They can be useful in certain scenarios (changing return values in defer functions, for example).

## Log levels

We want to be consistent regarding log levels used. We use the standard levels (`DEBUG`, `INFO`, `WARN`, `ERROR`, and `FATAL`). Following the code review document used as a base, we shouldn't use the `PANIC` level.

* `DEBUG`: As the name suggests, debug logs should be used to output information that is useful for debugging. These logs provide information useful for developing features, or fixing bugs. Because logging things like orders has a performance impact, we wrap log statements in an `if`, making sure we only call `log.Debug` if the log level is active.

```go
if log.Level() == logging.Debug {
    log.Debug("log entry here", logging.Order(order))
}
```

* `INFO`: These logs should be informative in the sense that they indicate that the application is working as intended, and something significant has happened. Examples of this are: the core is closing out distressed traders, a market is entering/exiting auction mode, a new market was added, etc...
* `WARN`: These logs indicate that something unusual has happened, but it's expected behaviour all the same. For example: a market doesn't have sufficient information to generate an accurate close-out price (used in position resolution), settlements are having to draw on the insurance pool to pay out traders, or even: the market has had to enter to loss socialisation flow (insufficient funds in the insurance pool).
* `ERROR`: Error logs indicate that the core was unable to perform the expected logic: the core tried to close out traders, but there weren't enough entries in the order book to match the network position (the net position taken over by the network from distressed traders). Depending on what exactly happened, a market might have to fall back on an _"emergency procedure"_ like entering auction mode, or suspending a market. Error logs indicate something went wrong, and we've had to handle the situation in a specific way. What we do is still defined behaviour, but really: these situations shouldn't have happened in the first place. The core can still continue to do its job, but a particular command resulted in an error. For example: a trader submitted an order, but the order couldn't be accepted because the trader didn't have the required margin, or doesn't have a general account for the right asset.
* `FATAL`: These logs indicate that the node was unable to carry on. Something went terribly wrong, and this is likely due to a bug. Immediate investigation and fixing is required.
* `PANIC`: Simple: don't panic. This should either be a `FATAL` log, or an `ERROR`, depending on the case.

Notable exception: A context with timeout/cancel always returns an error if the context is cancelled (whether it be due to the time limit being reached, or the context being cancelled manually). These errors specify why a context was cancelled. This information is returned by the `ctx.Err()` function, but this should *not* be logged as an error. We either log this at the `INFO` or `DEBUG` level. When the context for a (gRPC) stream is cancelled, for example, we should either ignore the error, or log it at the `DEBUG` level.
The reason we might want to log this could be: to ensure that streams are closed correctly if the client disconnects, or the stream hasn't been read in a while. This information is useful when debugging the application, but should not be spamming the logs with `ERROR` entries: this is expected behaviour after all.
