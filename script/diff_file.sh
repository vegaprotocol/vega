#!/bin/bash

# Diff a file between one branch and another.

syntax() {
	extra="${1:-}"
	if test -n "$extra" ; then
		echo "Error: $extra"
		echo
	fi
	echo "Syntax: $0 gitlab_api_url gitlab_api_token gitlab_project_id branch1 branch2 filename"
	echo
	echo "gitlab_api_url:    Probably https://gitlab.com/api/v4"
	echo "gitlab_api_token:  20-character Gitlab token"
	echo "gitlab_project_id: Numeric Gitlab project ID"
	echo "branch1:           A branch name"
	echo "branch2:           Another branch name"
	echo "filename:          The file to diff, e.g. gateway/graphql/schema.graphql"
}

for app in base64 curl diff jq python3 ; do
	if ! command -v "$app" 1>/dev/null ; then
		echo "Error: Need program: $app"
		exit 1
	fi
done

gitlab_api="${1:-}"
if test -z "$gitlab_api" ; then
	syntax "Need a Gitlab API URL"
	exit 1
fi

gitlab_api_token="${2:-}"
if test -z "$gitlab_api_token" ; then
	syntax "Need a Gitlab API token"
	exit 1
fi
auth_header="PRIVATE-TOKEN: ${gitlab_api_token}"

project_id="${3:-}"
if test -z "$project_id" ; then
	syntax "Need a Gitlab project ID"
	exit 1
fi

branch1="${4:-}"
branch2="${5:-}"
filename="${6:-}"
filename_urlquoted="$(python3 -c 'import sys,urllib.parse; print(urllib.parse.quote_plus(sys.argv[1]))' "$filename")"

get_file_for_branch() {
	# Api endpoint: GET /projects/:id/repository/files/:file_path
	# Doc: https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
	branch="$1"
	response_headers_file="$(mktemp)"
	file_json="$(curl --silent -D "$response_headers_file" --header "$auth_header" "$gitlab_api/projects/$project_id/repository/files/$filename_urlquoted?ref=$branch")"
	response_line="$(head -n1 <"$response_headers_file")"
	rm -f "$response_headers_file"
	if ! echo -n "$response_line" | grep -q '^HTTP/[0-9][.0-9]* 200 OK' ; then
		echo "Error: Failed to get file"
		echo "Response: $response_line"
		return
	fi
	echo "$file_json"
}

file1data="$(get_file_for_branch "$branch1")"
if echo "$file1data" | grep -q '^Error' ; then
	echo "$file1data"
	exit 1
fi

file2data="$(get_file_for_branch "$branch2")"
if echo "$file2data" | grep -q '^Error' ; then
	echo "$file2data"
	exit 1
fi

file1sha="$(echo "$file1data" | jq -r .content_sha256)"
file2sha="$(echo "$file2data" | jq -r .content_sha256)"

if test "$file1sha" == "$file2sha" ; then
	echo "Files match (SHA $file1sha)"
	exit 0
fi

file1tmpname="$(mktemp)"
file2tmpname="$(mktemp)"

echo "$file1data" | jq -r .content | base64 --decode >"$file1tmpname"
echo "$file2data" | jq -r .content | base64 --decode >"$file2tmpname"

diff -u --label "$filename $branch1" --label "$filename $branch2" "$file1tmpname" "$file2tmpname"
rm -f "$file1tmpname" "$file2tmpname"
