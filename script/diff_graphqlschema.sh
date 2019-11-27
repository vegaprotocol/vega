#!/bin/bash

if test -z "${CI:-}" ; then
	# Not in CI
	branch2="${1:-}"
	if test -z "$branch2" ; then
		echo "Syntax: $0 ref"
		echo "where 'ref' is a branch or tag name known to git."
		exit 1
	fi
else
	if test -n "${GITLAB_CI:-}" ; then
		# In Gitlab: https://docs.gitlab.com/ce/ci/variables/predefined_variables.html
		branch2="${CI_COMMIT_REF_NAME:-}"

		if test -z "$branch2" ; then
			echo "Failed to detect GitLab branch or tag."
			exit 1
		fi
	elif test -n "${DRONE:-}" ; then
		# In Drone: https://docker-runner.docs.drone.io/configuration/environment/variables/
		if test -n "${DRONE_PULL_REQUEST}" ; then
			# This is a merge request.
			branch1="${DRONE_TARGET_BRANCH}"
			branch2="${DRONE_SOURCE_BRANCH}"
		else
			# Commit
			branch2="${CI_COMMIT_BRANCH:-}"
			if test -z "$branch2" ; then
				branch2="${DRONE_TAG:-}"
			fi
		fi

		if test -z "$branch2" ; then
			echo "Failed to detect Drone branch or tag."
			exit 1
		fi
	else
		# In an unknown CI
		echo "Unknown CI"
		exit 1
	fi
fi

if test -z "$branch1" ; then
	# Deduce target branch
	case "$branch2" in
		develop)
			branch1="" # don't diff against master, it gets noisy on Slack#uidev.
			;;
		master)
			branch1=""
			;;
		release/v*)
			branch1=master
			;;
		*)
			branch1=develop
			;;
	esac
fi

code=0
if test -n "$branch1" ; then
	token="${GITLAB_API_TOKEN:-}"
	if test -z "$token" ; then
		echo "Need env var GITLAB_API_TOKEN"
		exit 1
	fi
	pushd "$(realpath "$(dirname "$0")/..")" 1>/dev/null || exit 1
	outputfile="$(mktemp)"
	bash script/diff_file.sh \
		"https://gitlab.com/api/v4" \
		"$token" \
		5726034 \
		"$branch1" \
		"$branch2" \
		"gateway/graphql/schema.graphql" \
		1>"$outputfile" 2>&1
	code="$?"
	if test "$code" == 0 ; then
		if grep -q '^---' "$outputfile" ; then
			code=1
			if test "${CI:-}" == "true" ; then
				cat "$outputfile"
				echo "Sending slack notification"
				slack_notify "#uidev" ":thinking-face:" "Heads up: GraphQL schema differs between \`$branch1\` and \`$branch2\`\\n\`\`\`\\n$(cat "$outputfile")\`\`\`"
			fi
		fi
	else
		cat "$outputfile"
	fi
	rm -f "$outputfile"
	popd 1>/dev/null || exit 1
fi
exit "$code"
