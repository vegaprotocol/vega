#!/bin/bash

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

gen_json
gen_docs
