#!/bin/bash -e

# When running on Netlify, can only use tools mentioned in:
# https://github.com/netlify/build-image/blob/xenial/included_software.md

generate_graphql() {
	echo TBD
}

generate_grpc() {
	grpc_dir="$dest_dir/grpc"
	mkdir "$grpc_dir"
	cp -a proto/doc/index.html "$grpc_dir/"
}

generate_rest() {
	rest_dir="$dest_dir/rest"
	mkdir -p "$rest_dir"
	cp -a proto/api/trading.swagger.json "$rest_dir/swagger.json"

	cat >"$rest_dir/index.html" <<EOZ
<!DOCTYPE html>
<html>
<head>
<title>REST API Documentation</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body { margin: 0; padding: 0; }
</style>
</head>
<body>
<redoc spec-url='/rest/swagger.json'></redoc>
<script src="https://rebilly.github.io/ReDoc/releases/latest/redoc.min.js"> </script>
</body>
</html>
EOZ
}

# # #

dest_dir=apidoc/public

rm -rf "$dest_dir"
mkdir -p "$dest_dir"

generate_grpc
generate_rest
