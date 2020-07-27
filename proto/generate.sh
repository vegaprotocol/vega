#!/bin/bash -e

# --go_out:            Generate *.pb.go
# --govalidators_out:  Generate *.validator.pb.go
# --grpc-gateway_out:  Generate *.pb.gw.go
# --swagger_out:       Generate *.swagger.json
# --doc_out:           Generate documentation in proto/doc/
# --doc_opt:           Options for generating documentation

paths="paths=source_relative"

# proto (excl subdirs): Generate *.pb.go and *.validator.pb.go
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

# Comment things before generating docs (#726, #1674)
# patch -p0 <proto/comment_undocumented.patch >/dev/null

mkdir -p proto/doc
protofiles="$(find ./proto/ -name '*.proto' -print | sort)"
echo -e 'html html\nmarkdown md' | while read -r fileformat fileextension
do
	outputfile="proto/doc/index.$fileextension"
	rm -f "$outputfile"
	echo -n "$protofiles" \
		| xargs protoc \
			-I. \
			-Iproto \
			-Ivendor \
			-Ivendor/github.com/google/protobuf/src \
			--doc_out="$(dirname "$outputfile")" \
			--doc_opt="$fileformat,$(basename "$outputfile")"
	sed --in-place -e 's#[ \t][ \t]*$##' "$outputfile"
done

# shellcheck disable=SC2016
sed --in-place -r \
	-e 's#`([^`]*)`#<tt>\1</tt>#g' \
	-e 's#\[([^]]*)\]\(([^)]*)\)#<a href="\2">\1</a>#g' \
	proto/doc/index.html

# proto/api: Generate *.swagger.json
grpc_api_configuration="grpc_api_configuration=gateway/rest/grpc-rest-bindings.yml"
find proto/api -maxdepth 1 -name '*.proto' | sort | while read -r protofile
do
	protoc \
		-I. \
		-Iproto \
		-Ivendor \
		-Ivendor/github.com/google/protobuf/src \
		--swagger_out="logtostderr=true,$grpc_api_configuration:." \
		"$protofile"
done

# Uncomment things after generating docs
#patch --reverse -p0 <proto/comment_undocumented.patch >/dev/null
find proto -name '*.proto.orig' -exec rm '{}' ';'

# Generate *.validator.pb.go, *.pb.gw.go
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
		"$protofile"
done

# Make *.validator.pb.go files deterministic.
find proto -name '*.validator.pb.go' | sort | while read -r pbfile
do
	sed -i -re 's/this\.Size_/this.Size/' "$pbfile" \
		&& ./script/fix_imports.sh "$pbfile"
done

chmod 0644 proto/*.go proto/api/*.go
