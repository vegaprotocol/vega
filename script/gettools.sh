#!/bin/bash -e

# Note: Make sure the versions match the ones in devops-infra/docker/cipipeline/Dockerfile
BUF_VER="1.0.0-rc12" # do not add "v" prefix


gettools_develop() {
	# These are all the tools required to develop trading-core.

	# tools = "golocation@version"
	tools="github.com/bufbuild/buf/cmd/buf@v$BUF_VER
google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.7.3
github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.7.3
github.com/gordonklaus/ineffassign@v0.0.0-20190601041439-ed7b1b5ee0f8
golang.org/x/lint/golint
github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1
golang.org/x/tools/cmd/goimports@v0.0.0-20190329200012-0ec5c269d481
honnef.co/go/tools/cmd/staticcheck@2019.2.3"
	# Note: Make sure the above tools and versions match the ones in devops-infra/docker/cipipeline/Dockerfile
	echo "$tools" | while read -r toolurl ; do
		go install "$toolurl"
	done
}

# # #

case "$1" in
develop)
	gettools_develop
	;;
*)
	echo "Syntax: $0 {develop}"
	exit 1
	;;
esac

echo "All ok."
