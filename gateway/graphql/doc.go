// Package gql contains code for running the GraphQL-to-gRPC gateway.
//
// In order to add a new GraphQL endpoint, add an entry to either the
// `Mutation`, `Query` or `Subscription` sections of
// `gateway/graphql/schema.graphql`. Example:
//
//     # SomeNewEndpoint does something
//     somenewendpoint(
//       # somestring
//       someStr: String!,
//       # someint
//       someInt: Int!
//     ): SomeNewEndpointResponse!
//
//     type SomeNewEndpointResponse {
//       someAnswer: String!
//       someStringList: [String!]
//     }
//
// Then run `make gqlgen`.
//
// Your new endpoint above will require a `SomeNewEndpointRequest` and `SomeNewEndpointResponse` message to be defined in the trading.proto file.
// Once this is defined you can run `make proto` to generate the structures required to add the resolvers below.
// e.g.
//	message SomeNewEndpointRequest {
//	  string orderID = 1;
//	  string referenceID = 2;
//  }
//
//  message SomeNewEndpointResponse {
//	vega.Order order = 1;
//  }
//
// Also a function definition needs to be defined in the trading.proto to show the parameters and return strutures for the new function
// e.g. rpc SomeNewEndpoint (SomeNewEndpointRequest) returns (SomeNewEndpointResponse)
//
// Next, in `gateway/graphql/resolvers.go`, add the endpoint to the
// `TradingClient` interface if the new endpoint is a mutation, else add it to TradingDataClient if is it just a query,
// and add a function implementation, using the
// function definition from `generated.go`. Example:
//
//     type TradingClient interface {
//         // ...
//         SomeNewEndpoint(context.Context, *api.SomeNewEndpointRequest, ...grpc.CallOption) (*api.SomeNewEndpointResponse, error)
//         // ...
//     }
//
//     // <<MQS>> is one of: Mutation, Query, Subscription
//     func (r *My<<MQS>>Resolver) SomeNewEndpoint(ctx context.Context, someStr string, someInt int64) (*SomeNewEndpointResponse, error) {
//         req := &protoapi.SomeNewEndpointRequest{
//             // ...
//         }
//
//         response, err := r.tradingClient.SomeNewEndpoint(ctx, req)
//         if err != nil {
//             return nil, err
//         }
//
//         return &SomeNewEndpointResponse{/* ... */}, nil
//     }
//
// Now add the new function to the `trading.go` or `trading_data.go` package to actually perform the work
//
// Lastly, make sure mocks are up to date, then run tests: `make mocks test`
package gql
