# Coding style

This document serves to outline the coding style standard we aim to follow throughout the codebase.

## Starting point

As a starting point, we should follow [the official Golang `CodeReviewComments` document](https://github.com/golang/go/wiki/CodeReviewComments). The basics are:

* names are `camelCased` or `CamelCased`, for functions, types, and constants alike.
* Avoid stutter (`markets.NewMarket` should be `markets.New()` etc...)
* Code should be passed though the `gofmt` tool
* ...

## Superset

As a basis, we're using the Golang `CodeReviewComments`. We're adding a few things to that that either have proven to be issues in our codebase, or things that are considered _good practice_, and we therefore want to enforce.

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

5. Type mapping: It's a common thing to need to map one type onto another (e.g. `log.DebugLevel` mapped onto a string representation). We use protobuf types throughout, many of which contain `oneof` fields. When we assign these values, we use a `switch` statement. This denotes that the code maps a `oneof` because:
    a) Only one case applies
    b) A switch performs better. The more `if`'s are needed, the slower the mapping becomes.
    c) The use of `if`'s and `else`'s makes the code look more complex than it really is. `if-else` hint at more complex logic (checking if a map contains an entry, checking for errors, etc...). Type mapping is just a matter of extracting the typed data, and assigning it.

Compare the following:

```go
func ifMap(assignTo T, oneOf Tb) {
    if a := oneOf.A(); a != nil {
        assignTo.A = a
    } else if b := oneOf.B(); b != nil {
        assignTo.B = b
    } else {
        return ErrUnmappableType
    }
}

func switchMap(assignTo T, oneOf Tb) {
    switch t := oneOf.Field.(type) {
    case *A:
        assignTo.A = t
    case *B:
        assignTo.B = t
    default:
        return ErrUnmappableType
    }
}
```

The latter not only looks cleaner, it results in fewer function calls (the `if` equivalent will call all getters until a non-nil value is returned), it's easier to maintain (adding another value is a 2 line change), and clearly communicates what this function does. Instantly, anyone looking at this code can tell that there's no business logic involved.

6. Return early whenever possible.

## Unit tests

Whenever implementing a new feature, new unit tests will have to be written. Unit tests, by definition, should use mocks rather than actual dependencies. We generate mocks for interfaces per package with a simple `//go:generate` command:

```go
//go:generate go run github.com/golang/mock/mockgen -destination mocks/some_dependency_mock.go -package mocks code.vegaprotocol.io/vega/pkg SomeDependency
type SomeDependency interface {
    DoFoo() error
    DoBar(ctx context.Context, ids []string) ([]*Bar, error)
}
```

From this, it ought to be clear that mocks are generated per-package (including in cases where several packages depend on a single object implementing the interface they need). Mock files are written to a sub-package/directory of the package we're testing: `mocks`. Generated files have the `_mock` suffix in their name.

The unit tests themselves sit next to the package files they cover, preferably with the same name + `_test` suffix (so `engine.go` tests in `engine_test.go`).
The test file itself also adds the `_test` suffix to the package name, effectively running tests in a different package. This ensures we're testing only the exported API. Covering unexported functions shouldn't be a problem. If an unexported function cannot be covered through calls made to the exported API, then it's dead code and can be removed.

Tests should be grouped in terms of what they cover. Each group ideally contains a simple scenario (happy path), a failure, and a couple more complex scenario's. Taking the collateral engine as an example, we see test grouping like this:


```go
func TestCollateralTransfer(t *testing.T) {}

func TestCollateralMarkToMarket(t *testing.T) {}

func TestAddTraderToMarket(t *testing.T) {}

func TestRemoveDistressed(t *testing.T) {}

func TestMarginUpdateOnOrder(t *testing.T) {}

func TestTokenAccounts(t *testing.T) {}

func TestEnableAssets(t *testing.T) {}

func TestCollateralContinuousTradingFeeTransfer(t *testing.T) {}
```

Each main test function contains a number of `t.Run("brief description of the specific test", testSpecificCaseFunc)` statements. The advantage is that running `go test -v ./collateral/...` groups the output per functionality, listing each specific test case, and whether or not it succeeded. Opening the file, jumping to a test group and locating the specific test case is faster to do than to filter through the same number of tests in a single file without any grouping applied to them.

It's also easier on reviewers to look at a PR and find a new test group when a new feature is added. If no such group is found, then it's pretty obvious no new tests were added. If a new group is added, we can see in a single function what scenario's have a unit test covering them.

When changes to specs, or internal implementations of existing features happen, these groups aid in refactoring. If the MarkToMarket transfers change in whatever way, we should be able to get the tests to pass simply by updating the `TestCollateralMarkToMarket` group.

## Protobuf

In addition to the Golang code review standards, we want to be consistent:

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

* `DEBUG`: As the name suggests, debug logs should be used to output information that is **useful to core developers** for debugging. These logs provide information useful for developing features, or fixing bugs. Because logging things like orders has a performance impact, we wrap log statements in an `if`, making sure we only call `log.Debug` if the log level is active.

  ```go
  if log.Level() == logging.Debug {
      log.Debug("log entry here", logging.Order(order))
  }
  ```
* `INFO`: These logs should be **informative to node operators** in the sense that they indicate that the application is working as intended, and something significant has happened. Messages should be one-off (e.g. at start-up or shutdown) or occasional (e.g. market created or settled). Do not log at `INFO` level inside loops, or in a way which means that increased activity causes a proportional increase in the number of log messages (e.g. a distressed trader was closed out, a market entered/exited auction mode).
* `WARN`: These logs indicate that something unusual (but expected) has happened, the node is now operating in a sub-optimal way, and the node operator could do something to fix this to remove so that the log message would not appear.
* `ERROR`: These logs indicate that there was a problem with a non-instrumental subsystem (e.g. the REST HTTP server died) but the node can continue, albeit in a degraded state (e.g. gRPC and GraphQL are fine, but not the dead REST HTTP server). The node operator probably needs to take some significant action (e.g. restarting the node, augmenting node hardware).
* `FATAL`: These logs indicate that the node was unable to continue. Something went terribly wrong, and this is likely due to a bug. Immediate investigation and fixing is required. `os.Exit(1)` is called, which does not run deferred functions.
* `PANIC`: Simple: don't panic. This should either be a `FATAL` log, or an `ERROR`, depending on the case.

Notable exception: A context with timeout/cancel always returns an error if the context is cancelled (whether it be due to the time limit being reached, or the context being cancelled manually). These errors specify why a context was cancelled. This information is returned by the `ctx.Err()` function, but this should *not* be logged as an error. We log this at the `DEBUG` level. When the context for a (gRPC) stream is cancelled, for example, we should either ignore the error, or log it at the `DEBUG` level.
The reason we might want to log this could be: to ensure that streams are closed correctly if the client disconnects, or the stream hasn't been read in a while. This information is useful when debugging the application, but should not be spamming the logs with `ERROR` entries: this is expected behaviour after all.

## API response errors

The audience for API responses is different to the audience for log messages. An API user who submits a message and receives an error response is interested in what they can do to fix their message. They are not interested in core code (e.g. stack traces, file references or line numbers) or in the node (e.g. disk full, failed to write to badger store).

## Helpful errors

Errors returned from functions should be as helpful as possible, for example by including function parameters.

Example:

```go
func DoAllThings(ids []string) error {
    for _, id := range ids {
        err := DoSomething(id)
        if err != nil {
            return err
        }
    }
}

func DoSomething(id string) error {
    err := doSomeSub1Thing(id)
    if err != nil {
        // details from err are lost, and there is no mention of "id".
        return ErrFailedToDoSomeSub1Thing
    }

    err = doSomeSub2Thing(id)
    if err != nil {
        // details from err are included, but there is still no mention of "id".
        return fmt.Errorf("error doing some sub2 thing: %v", err)
    }

    // ...
}
```

The omission of the identifier `id` means that we don't know which call to `DoSomething` was the one that caused the error.

## Inappropriate wording
Some of the wording that was used as a standard 10 years ago is no long considered correct for use in open source software. We should use the updated version of these naming schemes in all of our code and documentation.

* Blacklist/Whitelist -> Denylist/Allowlist
* Master/Slave -> Primary/Replica
