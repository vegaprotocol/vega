// Package rest contains code for running the REST-to-gRPC gateway.
//
// In order to add a new REST endpoint, add an entry to
// `interal/gateway/rest/grpc-rest-bindings.yml`.
//
// Run `make proto` to generate (amonog others) the swagger json file.
//
// Run `make rest_check` to make sure that all the endpoints in the bindings
// yaml file make it into the swagger json file.
package rest
