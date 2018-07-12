#!/usr/bin/env bash

# Set GOPATH if it's unset
#if [ -z ${var+x} ]
#then
#    GOPATH=$HOME/go
#    echo "Using ${GOPATH} for GOPATH"
#fi

SERVICE_HOME=api
PROTO_DEF=$SERVICE_HOME/service.proto

# This first protoc command creates a gRPC stub:
# - All the protocol buffer code to populate, serialize, and retrieve our request and response message
# - An interface type (or stub) for clients to call with the methods defined in the RouteGuide service.
# - An interface type for servers to implement, also with the methods defined in the RouteGuide service.
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --go_out=plugins=grpc:. \
  $PROTO_DEF

#  -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \

# This creates a reverse proxy to forward HTTP requests into gRPC requests
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --grpc-gateway_out=logtostderr=true,grpc_api_configuration=$SERVICE_HOME/rest-bindings.yml:. \
  $PROTO_DEF

#  -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \

# Generates Swagger documentation
protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  --swagger_out=logtostderr=true,grpc_api_configuration=$SERVICE_HOME/rest-bindings.yml:. \
  $PROTO_DEF

#  -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
