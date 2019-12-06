#!/bin/bash -e

# --go_out:            Generate *.pb.go
# --govalidators_out:  Generate *.validator.pb.go
# --grpc-gateway_out:  Generate *.pb.gw.go
# --swagger_out:       Generate *.swagger.json
# --doc_out:           Generate documentation in proto/doc/
# --doc_opt:           Options for generating documentation

paths="paths=source_relative"

# Generate *.pb.go and *.validator.pb.go
find proto -maxdepth 1 -name '*.proto' | sort | while read -r protofile
do
	protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		--go_out=plugins="grpc,$paths:." \
		--govalidators_out="$paths:." \
		"$protofile"
done

# Generate proto/doc/
mkdir -p proto/doc
rm -f proto/doc/index.md
find ./proto/ -name '*.proto' -print0 \
	| sort -z \
	| xargs -0 protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		--doc_out=proto/doc \
		--doc_opt=markdown,index.md

sed -i -e 's#[ \t][ \t]*$##' proto/doc/index.md

# Generate *.pb.gw.go and *.swagger.json
grpc_api_configuration="grpc_api_configuration=gateway/rest/grpc-rest-bindings.yml"
find proto/api -maxdepth 1 -name '*.proto' | sort | while read -r protofile
do
	protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		--go_out="plugins=grpc,$paths:." \
		--govalidators_out="$paths:." \
		--grpc-gateway_out="logtostderr=true,$grpc_api_configuration,$paths:." \
		--swagger_out="logtostderr=true,$grpc_api_configuration:." \
		"$protofile"
done

# Make *.validator.pb.go files deterministic.
find proto -name '*.validator.pb.go' | sort | while read -r pbfile
do
	sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
done
