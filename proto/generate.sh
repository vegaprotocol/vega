#!/bin/bash -e

# --go_out:            Generate *.pb.go
# --govalidators_out:  Generate *.validator.pb.go
# --grpc-gateway_out:  Generate *.pb.gw.go
# --swagger_out:       Generate *.swagger.json

find proto -maxdepth 1 -name '*.proto' | sort | while read -r protofile
do
	protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		-Ivendor/github.com/grpc-ecosystem/grpc-gateway \
		-Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc,paths=source_relative:. \
		--govalidators_out=paths=source_relative:. \
		"$protofile"
done

find proto/api -maxdepth 1 -name '*.proto' | sort | while read -r protofile
do
	protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		-Ivendor/github.com/grpc-ecosystem/grpc-gateway \
		-Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc,paths=source_relative:. \
		--govalidators_out=paths=source_relative:. \
		--grpc-gateway_out=logtostderr=true,paths=source_relative:. \
		--swagger_out=logtostderr=true:. \
		"$protofile"
done

# Make *.validator.pb.go files deterministic.
find proto -name '*.validator.pb.go' | sort | while read -r pbfile
do
	sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
done
