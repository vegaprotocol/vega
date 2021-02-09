#!/bin/bash

TARGET="proto"


function check() {
	if [[ ! -d "$TARGET" ]]; then
		echo "Target directory \`$TARGET\` not found, run this script from Vega's repository root path"
		exit 1
	fi
}

function gen_code() {
	# generate code, grpc and validators code
	buf generate

	# Since ./proto/github/{grpc-ecosystem,mwitkow} are dependencies,
	# buf will generate code for them to
	rm -rf ./proto/github.com

	# Make *.validator.pb.go files deterministic.
	find proto -name '*.validator.pb.go' | sort | while read -r pbfile
	do
        sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
	done

	chmod 0644 proto/*.go proto/api/*.go
}

function gen_docs() {
	buf generate --template=./proto/buf.gen.doc.yaml # generate docs
	buf generate --path=./proto/api --template=./proto/api/buf.gen.yaml # generate swagger
}


check
gen_code
gen_docs
