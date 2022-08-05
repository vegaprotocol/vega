#!/bin/bash

# go to the repo root directory
cd "$(dirname "$0")/.."

function clean() {
  rm -rf protos/generated
  rm -rf protos/swagger
}

function gen_json() {
	mkdir -p protos/generated/json/vega
	mkdir -p ./protos/generated/json/data-node/api/v1
	mkdir -p ./protos/generated/json/data-node/api/v2

	protoc --experimental_allow_proto3_optional --jsonschema_out=./protos/generated/json/vega --proto_path=./protos/sources protos/sources/vega/*.proto
	protoc --experimental_allow_proto3_optional --jsonschema_out=./protos/generated/json/data-node/api/v1 --proto_path=./protos/sources protos/sources/data-node/api/v1/*.proto
	protoc --experimental_allow_proto3_optional --jsonschema_out=./protos/generated/json/data-node/api/v2 --proto_path=./protos/sources protos/sources/data-node/api/v2/*.proto
}

function gen_docs() {
  mkdir -p ./protos/generated

  protoc --experimental_allow_proto3_optional --doc_out=./protos/generated --doc_opt=json,proto.json --proto_path=protos/sources/ \
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

function gen_swagger() {
	buf generate --path=./protos/sources/vega/api --template=./protos/sources/vega/api/v1/buf.gen.yaml --output ./protos # generate swagger
	buf generate --path=./protos/sources/data-node/api/v1 --template=./protos/sources/data-node/api/v1/buf.gen.yaml --output ./protos # generate swagger
}

clean
gen_swagger
gen_json
gen_docs
