# How to add a new gRPC endpoint

Add `Request` and `Response` messages in `proto/api/trading.proto`: 

```proto
message SomeNewEndpointRequest {
  string somestr = 1;
  int64 someint = 2;

}

message SomeNewEndpointResponse {
  string someanswer = 1;
  repeated string somestringlist = 2;
}
```

Add a function to `internal/api/somefile.go`:

```go
func (s *tradingService) SomeNewEndpoint(
	ctx context.Context, req *protoapi.SomeNewEndpointRequest,
) (*protoapi.SomeNewEndpointResponse, error) {
	/* Do stuff */
	return &protoapi.SomeNewEndpointResponse{/* ... */}, nil
}
```

## GraphQL

Add the endpoint to `internal/gateway/graphql/schema.graphql`.

TBC

Run `make gqlgen`.


## REST
Add the endpoint to `interal/gateway/rest/grpc-rest-bindings.yml`.

Run `make proto` to generate (amonog others) the swagger json file.

Run `make rest_check` to make sure that all the endpoints in the bindings yaml file make it into the swagger json file.
