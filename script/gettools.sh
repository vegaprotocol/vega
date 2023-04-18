#!/bin/bash -e

tools="github.com/bufbuild/buf/cmd/buf@v1.17.0
google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema@1.3.7
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.7.3
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.7.3
github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.5.1
github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0
"

# Note: Make sure the above tools and versions match the ones in devops-infra/docker/cipipeline/Dockerfile
echo "$tools" | while read -r toolurl ; do
	go install "$toolurl"
done
