#!/usr/bin/env bash

SERVICE_HOME=api
PROTO_DEF=$SERVICE_HOME/grpc.proto

# This first protoc command creates a gRPC stub:
# - All the protocol buffer code to populate, serialize, and retrieve our request and response message
# - An interface type (or stub) for clients to call with the methods defined in the RouteGuide service.
# - An interface type for servers to implement, also with the methods defined in the RouteGuide service.
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --go_out=plugins=grpc:. \
  $PROTO_DEF

# This creates a reverse proxy to forward HTTP requests into gRPC requests
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --grpc-gateway_out=logtostderr=true,grpc_api_configuration=$SERVICE_HOME/grpc-rest-bindings.yml:. \
  $PROTO_DEF

# Generates Swagger documentation
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --swagger_out=logtostderr=true,grpc_api_configuration=$SERVICE_HOME/grpc-rest-bindings.yml:. \
  $PROTO_DEF
