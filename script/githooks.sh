#!/bin/bash -e

abort() {
	stage="${hook//[a-z]*-/}"
	echo "Aborting $stage"
	exit 1
}

go_format() {
	go_files_modified="$(git diff --cached --name-only | grep '\.go$' || true)"
	if test -z "$go_files_modified"
	then
		return
	fi

	if ! which gofmt 1>/dev/null
	then
		echo "Please install gofmt"
		abort
	fi

	result="$(echo "$go_files_modified" | xargs -r gofmt -l)"
	if test -n "$result" ; then
		echo "gofmt failed for:"
		echo "$result"
		abort
	fi
}

pre_commit() {
	go_format
}

pre_push() {
	# remote="$1"
	# url="$2"

	z40=0000000000000000000000000000000000000000

	makeprotochecks=""
	maketest=""
	# shellcheck disable=SC2034
	while read -r local_ref local_sha remote_ref remote_sha
	do
		if [ "$local_sha" = $z40 ]
		then
			# Handle delete
			:
		else
			if [ "$remote_sha" = $z40 ]
			then
				# New branch, examine all commits
				range="$local_sha"
			else
				# Update to existing branch, examine new commits
				range="$remote_sha..$local_sha"
			fi

			commit_hash="$(git rev-list -n1 "$range")"
			commit_info="$(git show --pretty=oneline --name-only "$commit_hash")"
			# commit_msg="$(echo "$commit_info" | head -n1 | cut -f2- -d ' ')"
			commit_files="$(echo "$commit_info" | tail -n +2 | cut -f2- -d ' ')"

			echo "$commit_files" | grep -q '\.proto$' && makeprotochecks=yes
			echo "$commit_files" | grep -q '\.go$' && maketest=yes
		fi
	done

	if test -n "$makeprotochecks"
	then
		echo "Proto files to be pushed. Running checks..."
		code=0
		for target in grpc_check proto_check
		do
			make "$target" || code=1
		done
		test "$code" -gt 0 && abort
	fi

	if test -n "$maketest"
	then
		echo "Go files to be pushed. Running checks..."
		make test || abort
	fi
}

# # #

exec 1>&2 # redirect stdout to stderr

hook="$(basename "$0")"
case "$hook" in
pre-commit)
	pre_commit
	;;
pre-push)
	pre_push "$@"
	;;
*)
	echo "Hook not implemented: $hook"
	abort
	;;
esac

exit 0
