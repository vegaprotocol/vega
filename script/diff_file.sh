#!/bin/bash

# Diff a file between one branch and another.

pushd "$(realpath "$(dirname "$0")")" 1>/dev/null || exit 1
# shellcheck disable=SC1091
source bash_functions.sh
popd 1>/dev/null || exit 1

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
	# Sample response:
	# {
	#   "file_name": "schema.graphql",
	#   "file_path": "gateway/graphql/schema.graphql",
	#   "size": 21215,
	#   "encoding": "base64",
	#   "content_sha256": "8e06...07ec",
	#   "ref": "develop",
	#   "blob_id": "d099...1f0a",
	#   "commit_id": "06c3...28e7",
	#   "last_commit_id": "06b1...84f1",
	#   "content": "IyMgVkVHQSAtIE...mVyYWwKfQo="
	# }

	branch="$1"

	response_headers_file="$(mktemp)"
	file_json="$(curl --silent -D "$response_headers_file" --header "$auth_header" "$gitlab_api/projects/$project_id/repository/files/$filename_urlquoted?ref=$branch")"
	response_line="$(head -n1 <"$response_headers_file")"
	rm -f "$response_headers_file"
	if ! echo -n "$response_line" | grep -q '^HTTP/[0-9][.0-9]* 200 OK' ; then
		echo "Error: Failed to get file for branch $branch: $filename"
		echo "Response code was: $response_line"
		exit 1
	fi
	echo "$file_json"
}

echo "Getting $filename on $branch1"
file1data="$(get_file_for_branch "$branch1")"
echo "Getting $filename on $branch2"
file2data="$(get_file_for_branch "$branch2")"

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

echo "Files differ between branch $branch1 and $branch2:"
diffname="$(mktemp)"
diff -u --label "$filename $branch1" --label "$filename $branch2" "$file1tmpname" "$file2tmpname" | tee "$diffname"

echo
echo "Latest commit: $(echo "$file1data" | jq .commit_id) (repo $branch1)"
echo "Latest commit: $(echo "$file2data" | jq .commit_id) (repo $branch2)"
echo
echo "Latest commit: $(echo "$file1data" | jq .last_commit_id) ($filename $branch1)"
echo "Latest commit: $(echo "$file2data" | jq .last_commit_id) ($filename $branch2)"

gitlab_ci="${GITLAB_CI:-false}"
if test "$gitlab_ci" == "true" ; then
	echo "Sending slack notification"
	pipeline_url="${CI_PIPELINE_URL:-[failed to get pipeline URL]}"
	slack_notify "#uidev" ":thinking-face:" "Heads up: GraphQL schema differs between \`$branch1\` and \`$branch2\` (see \`autogen_checks\` from $pipeline_url for details)\\n\`\`\`\\n$(cat "$diffname")\`\`\`"
fi
rm -f "$file1tmpname" "$file2tmpname" "$diffname"
