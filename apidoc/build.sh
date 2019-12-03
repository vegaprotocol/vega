#!/bin/bash -e

# When running on Netlify, can only use tools mentioned in:
# https://github.com/netlify/build-image/blob/xenial/included_software.md

generate_index() {
	cat >"$dest_dir/index.html" <<EOZ
<!DOCTYPE html>
<html>
<head>
<title>Vega API Documentation</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body { margin: 0; padding: 0; }
</style>
</head>
<body>
<ul>
<li><a href="/grpc/">gRPC</a></li>
<li><a href="/graphql/">GraphQL</a></li>
<li><a href="/rest/">REST</a></li>
</ul>
</body>
</html>
EOZ
}

generate_graphql() {
	graphql_dir="$dest_dir/graphql"
	mkdir -p "$graphql_dir"
	pushd apidoc 1>/dev/null || exit 1
	yarn install
	yarn build
	popd 1>/dev/null || exit 1
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

generate_index

generate_graphql
generate_grpc
generate_rest
