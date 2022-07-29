#!/bin/bash

function gen_code() {
	buf generate
	if [ $? -ne 0 ]; then
		exit $?
	fi
}

function gen_swagger() {
	buf generate --path=./protos/sources/vega/api --template=./protos/sources/vega/api/v1/buf.gen.yaml # generate swagger
	buf generate --path=./protos/sources/data-node/api/v1 --template=./protos/sources/data-node/api/v1/buf.gen.yaml # generate swagger
}

gen_code
gen_swagger
