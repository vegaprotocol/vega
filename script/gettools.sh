#!/bin/bash -e

# Note: Make sure the versions match the ones in devops-infra/docker/cipipeline/Dockerfile
PROTOC_VER="3.7.1" # do not add "v" prefix
PROTOC_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VER}/protoc-${PROTOC_VER}-linux-x86_64.zip"
PROTOBUF_VER="1.3.1" # do not add "v" prefix

check_protoc() {
	echo "Checking for existance: protoc"
	if ! which protoc 1>/dev/null ; then \
		echo "Not found on \$PATH': protoc" >/dev/stderr
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

check_protoc_gen_go() {
	echo "Checking for existance: protoc-gen-go"
	protobuf_base="github.com/golang/protobuf"
	if ! which protoc-gen-go 1>/dev/null ; then \
		go get "$protobuf_base@v${PROTOBUF_VER}"
		go get "$protobuf_base/protoc-gen-go@v${PROTOBUF_VER}"
	fi
	if ! which protoc-gen-go 1>/dev/null ; then \
		echo "Either could not go get protoc-gen-go" >/dev/stderr
		echo "Or \$GOPATH is not on \$PATH" >/dev/stderr
		exit 1
	fi
	echo "Checking version: protoc-gen-go, ${PROTOBUF_VER}"
	protoc_gen_go="$(which protoc-gen-go)"
	# Note: Again, the dot chars in the version string are left unescaped.
	if ! strings "$protoc_gen_go" | grep -q "$protobuf_base"'\s'"v${PROTOBUF_VER}" ; then \
		echo "Could not find version string in protoc-gen-go" >/dev/stderr
		exit 1
	fi
}

# # #

check_protoc
check_protoc_gen_go
echo "All ok."
