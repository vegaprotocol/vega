// Package api contains code for running the gRPC server.
//
// In order to add a new gRPC endpoint, add proto content (rpc call, request
// and response messages), then add the endpoint function implementation in
// `internal/api/somefile.go`. Example:
//
//     func (s *tradingService) SomeNewEndpoint(
//         ctx context.Context, req *protoapi.SomeNewEndpointRequest,
//     ) (*protoapi.SomeNewEndpointResponse, error) {
//         /* Implementation goes here */
//         return &protoapi.SomeNewEndpointResponse{/* ... */}, nil
//     }
//
// Add a test for the newly created endpoint in `internal/api/trading_test.go`.
package api
