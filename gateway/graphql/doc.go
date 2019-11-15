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
//     typeSomeNewEndpointResponse {
//       someAnswer: String!
//       someStringList: [String!]
//     }
//
// Then run `make gqlgen`.
//
// Next, in `gateway/graphql/resolvers.go`, add the endpoint to the
// `TradingClient` interface, and add a function implementation, using the
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
// Lastly, make sure mocks are up to date, then run tests: `make mocks test`
package gql
