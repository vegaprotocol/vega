#!/bin/bash

TARGET="vega"

function check() {
	if [[ ! -d "$TARGET" ]]; then
		echo "Target directory \`$TARGET\` not found, run this script from vega protos's repository root path"
		exit 1
	fi
}

function gen_code() {
	# generate code, grpc and validators code
	buf generate

	if [ $? -ne 0 ]; then
		exit $?
	fi

	# Since ./proto/github/{grpc-ecosystem,mwitkow} are dependencies,
	# buf will generate code for them to
	rm -rf ./github.com

	# Make *.validator.pb.go files deterministic.
	find vega -name '*.validator.pb.go' | sort | while read -r pbfile
	do
        sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
	done
	find data-node -name '*.validator.pb.go' | sort | while read -r pbfile
	do
        sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
	done
}

function gen_swagger() {
	buf generate --path=./protos/sources/vega/api --template=./protos/sources/vega/api/v1/buf.gen.yaml # generate swagger
	buf generate --path=./protos/sources/data-node/api/v1 --template=./protos/sources/data-node/api/v1/buf.gen.yaml # generate swagger
}

function gen_json() {
	rm -rf protos/generated
	mkdir -p protos/generated/json/vega
	mkdir -p ./protos/generated/json/data-node/api/v1
	mkdir -p ./protos/generated/json/data-node/api/v2

	protoc --jsonschema_out=./protos/generated/json/vega --proto_path=./protos/sources protos/sources/vega/*.proto
	protoc --jsonschema_out=./protos/generated/json/data-node/api/v1 --proto_path=./protos/sources protos/sources/data-node/api/v1/*.proto
	protoc --jsonschema_out=./protos/generated/json/data-node/api/v2 --proto_path=./protos/sources protos/sources/data-node/api/v2/*.proto
}

function gen_docs() {
  mkdir -p generated

  protoc --doc_out=./protos/generated --doc_opt=json,proto.json --proto_path=protos/sources/ \
  protos/sources/vega/*.proto \
  protos/sources/vega/oracles/**/*.proto \
  protos/sources/vega/commands/**/*.proto \
  protos/sources/vega/events/**/*.proto \
  protos/sources/vega/api/**/*.proto \
  protos/sources/vega/checkpoint/**/*.proto \
  protos/sources/vega/snapshot/**/*.proto \
  protos/sources/vega/events/**/*.proto \
  protos/sources/vega/wallet/**/*.proto \
  protos/sources/data-node/api/**/*.proto
}

check
gen_code
gen_swagger
gen_json
gen_docs
