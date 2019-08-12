# How to add a new API endpoint (gRPC + GraphQL + REST)

In `proto/api/trading.proto`:
* Add the endpoint to the `trading` service
* Add `Request` and `Response` messages

```proto
service trading {
  // ...
  rpc SomeNewEndpoint(SomeNewEndpointRequest) returns (SomeNewEndpointResponse);
  // ...
}

// ...

message SomeNewEndpointRequest {
  string somestr = 1;
  int64 someint = 2;
}

message SomeNewEndpointResponse {
  string someanswer = 1;
  repeated string somestringlist = 2;
}
```

In `internal/api/somefile.go`:
* Add the endpoint function implementation

```go
func (s *tradingService) SomeNewEndpoint(
	ctx context.Context, req *protoapi.SomeNewEndpointRequest,
) (*protoapi.SomeNewEndpointResponse, error) {
	/* Do stuff */
	return &protoapi.SomeNewEndpointResponse{/* ... */}, nil
}
```

## GraphQL

In `internal/gateway/graphql/schema.graphql`:
* Add the endpoint to one of the following sections:
  * `Mutation`
  * `Query`
  * `Subscription`

```graphql
# SomeNewEndpoint does something
somenewendpoint(
  # somestring
  someStr: String!,
  # someint
  someInt: Int!
): SomeNewEndpointResponse!

typeSomeNewEndpointResponse {
  someAnswer: String!
  someStringList: [String!]
}
```

Run `make gqlgen`.

## REST
Add the endpoint to `interal/gateway/rest/grpc-rest-bindings.yml`.

Run `make proto` to generate (amonog others) the swagger json file.

Run `make rest_check` to make sure that all the endpoints in the bindings yaml file make it into the swagger json file.
