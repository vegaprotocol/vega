#!/bin/bash -e

# Note: Make sure the versions match the ones in devops-infra/docker/cipipeline/Dockerfile
PROTOC_VER="3.7.1" # do not add "v" prefix
PROTOC_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VER}/protoc-${PROTOC_VER}-linux-x86_64.zip"
PROTOBUF_VER="1.3.1" # do not add "v" prefix

check_gotools() {
	# tools = "binary:golocation@version"
	tools="github.com/golang/protobuf@v$PROTOBUF_VER
github.com/golang/protobuf/protoc-gen-go@v$PROTOBUF_VER
github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.8.5
github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.8.5
github.com/mwitkow/go-proto-validators/protoc-gen-govalidators@v0.0.0-20190212092829-1f388280e944
github.com/vegaprotocol/modvendor@v0.0.1
golang.org/x/lint/golint
golang.org/x/tools/cmd/goimports@v0.0.0-20190329200012-0ec5c269d481"
	# Note: Make sure the above tools and versions match the ones in devops-infra/docker/cipipeline/Dockerfile
	echo "$tools" | while read -r toolurl ; do
		go get "$toolurl"
	done
}

check_protoc() {
	echo "Checking for existance: protoc"
	if ! which protoc 1>/dev/null ; then \
		echo "Not found on \$PATH: protoc" >/dev/stderr
		echo "Please install it from ${PROTOC_URL}" >/dev/stderr
		echo "And put the protoc binary in a dir on \$PATH" >/dev/stderr
		exit 1
	fi
	echo "Checking version: protoc, ${PROTOC_VER}"
	# Note: the dot chars in the version string are left unescaped. Shouldn't be a problem for grep.
	protoc_ver="$(protoc --version)"
	if ! echo "$protoc_ver" |grep -q "^libprotoc ${PROTOC_VER}$" ; then \
		echo "Wrong version: $protoc_ver" >/dev/stderr
		echo "Please install version ${PROTOC_VER} from ${PROTOC_URL}" >/dev/stderr
		exit 1
	fi
}

# # #

check_gotools
check_protoc
echo "All ok."
