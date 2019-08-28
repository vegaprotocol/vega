#!/bin/bash

# Test this script with:
# docker run --rm -ti \
# 	-v "$HOME/containergo/pkg:/go/pkg" \
# 	-v "$HOME/go/src/vega:/go/src/project" \
# 	-w /go/src/project \
# 	--entrypoint /bin/bash \
# 	registry.gitlab.com/vega-protocol/devops-infra/cipipeline:1.11.5 \
# 	-c 'make install && ./.deploy.sh devnet vega "/go/bin/vega:/home/vega/current/:0755"'

# Required vars (or files named the same as the var plus ".tmp"):
# - "${NET}_DEPLOY_SSH_PRIVATE_KEY"
# - "${NET}_DEPLOY_HOSTS"
# - "${NET}_DEPLOY_SSH_KNOWN_HOSTS"

# Optional vars:
# - "$SLACK_HOOK_URL" for sending Slack notifications

pushd "$(realpath "$(dirname "$0")")" 1>/dev/null || exit 1
source bash_functions.sh
popd 1>/dev/null || exit 1

check_apps() {
	# Check required programs
	apps=( rsync scp ssh )
	for app in "${apps[@]}" ; do
		if ! which "$app" 1>/dev/null ; then
			failure "Program missing: $app"
		fi
	done
}

check_vars() {
	# Check required environment variables
	keyvar="${net_ucase}_DEPLOY_SSH_PRIVATE_KEY"
	hostsvar="${net_ucase}_DEPLOY_HOSTS"
	knownhostsvar="${net_ucase}_DEPLOY_SSH_KNOWN_HOSTS"
	vars=( "$keyvar" "$hostsvar" "$knownhostsvar" )
	for var in "${vars[@]}" ; do
		if test -z "${!var}" ; then
			if test -f "${var}.tmp" ; then
				eval "${var}"'="$(cat '"${var}"'.tmp)"'
			fi
			if test -z "${!var}" ; then
				failure "Variable missing: \$${var}"
			fi
		fi
	done
}

install_files() {
	for host in ${!hostsvar} ; do
		for filespec in "$@" ; do
			src="$(echo "$filespec" | cut -f1 -d: )"
			srcfile="$(basename "$src")"
			dstdir="$(echo "$filespec" | cut -f2 -d: )/"
			dstfullpath="$dstdir$srcfile"
			dstfullpath="${dstfullpath//\/\//\/}" # squash multiple slashes
			perm="$(echo "$filespec" | cut -f3 -d: )"

			echo "$host: $src -> $dstfullpath ($perm)"

			# Rename existing file
			# Note: $(date) is in single quotes so it is expanded on the remote host
			ssh "$sshuser@$host" \
				'test -f "'"$dstfullpath"'" && mv "'"$dstfullpath"'" "'"$dstfullpath"'-$(date "+%Y.%m.%d-%H.%M.%S")"'

			# Copy new file
			rsync -avz "$src" "$sshuser@$host:$dstdir"

			# Set file permissions
			ssh "$sshuser@$host" \
				'chmod "'"$perm"'" "'"$dstfullpath"'"'
		done
	done

}

nodeloop() {
	# Syntax: nodeloop "Doing a thing" '/usr/local' '/bin/thing' ' ; echo "Done"'
	# Arg 1: a message
	# Args 2-N: some command(s). Strings will be concatenated with no space in between.
	msg="$1" ; shift
	cmd="" ; while test -n "$1" ; do cmd="$cmd$1" ; shift ; done
	maxcode=0
	for host in ${!hostsvar}
	do
		echo "$host: $msg"
		# shellcheck disable=SC2029
		ssh "$sshuser@$host" "$cmd"
		code="$?"
		test "$code" -gt "$maxcode" && maxcode="$code"
	done
	if test "$maxcode" -gt 0 ; then
		failure "Failed to run commands on all nodes. Max exit code was $maxcode"
	fi
}

nukedata_tendermint() {
	nodeloop \
		"Resetting tendermint chain" \
		'sudo -iu vega tendermint unsafe_reset_all'
}

nukedata_vega() {
	nodeloop \
		"Deleting vega stores" \
		'sudo -iu vega /bin/bash -c "' \
			'cd ; rm -rf current/tmp/*store .vega/*store' \
		'"'
}

traders_action() {
	action="${1:-}"
	case "$action" in
	start|stop)
		response_headers_file="$(mktemp)"
		output_file="$(mktemp)"
		curl -D "$response_headers_file" --silent -XPUT "https://bots.vegaprotocol.io/$net/v2/traders?action=$action" 1>"$output_file" 2>&1
		response_line="$(head -n1 <"$response_headers_file")"
		if ! echo -n "$response_line" | grep -q '^HTTP/[0-9][.0-9]* 200' ; then
			echo "Warning: Bad response from go-trade-bot: $response_line"
			echo "Headers:"
			cat "$response_headers_file"
			echo "Response:"
			cat "$output_file"
			echo "Continuing with deployment..."
		fi
		rm -f "$response_headers_file" "$output_file"
		;;
	*)
		failure "Invalid action for go-trade-bot: $action"
	esac
}

json_escape() {
	echo -n "$1" | python -c 'import json,sys; print(json.dumps(sys.stdin.read()))'
}

ssh_setup() {
	eval "$(ssh-agent -s)"

	mkdir -p ~/.ssh
	chmod 0700 ~/.ssh

	# Save SSH key to file
	echo "${!keyvar}" >~/.ssh/id_rsa
	chmod 0600 ~/.ssh/id_rsa

	# Save SSH known hosts to file
	echo "${!knownhostsvar}" >~/.ssh/known_hosts
	chmod 0644 ~/.ssh/known_hosts
}

ssh_tidy() {
	rm -rf ~/.ssh
}

start_vega_tendermint() {
	nodeloop \
		"Starting vega and tendermint with SystemD" \
		'cd ; ./current/vega --version ; ' \
		'/home/vega/bin/tendermint version ; ' \
		'sudo systemctl daemon-reload ; ' \
		'sudo systemctl restart vega ; ' \
		'sleep 1 ; ' \
		'sudo systemctl restart tendermint'
}

stop_vega_tendermint() {
	nodeloop \
		"Stopping vega and tendermint processes with SystemD" \
		'cd ; ./current/vega --version ; ' \
		'/home/vega/bin/tendermint version ; ' \
		'sudo systemctl daemon-reload ; ' \
		'sudo systemctl stop vega ; ' \
		'sudo systemctl stop tendermint ; ' \
		'sudo killall vega tendermint 1>/dev/null 2>&1 ; ' \
		'true'
}

failure() {
	extra="${1:-no error message supplied}"
	echo "$extra" >/dev/stderr
	gitlab_ci="${GITLAB_CI:-false}"
	if test "$gitlab_ci" == "true" ; then
		pipeline_url="${CI_PIPELINE_URL:-[failed to get piepline URL]}"
		msg="Failed to deploy to \`$net\`: $extra. See $pipeline_url for details."
	else
		msg="Failed to deploy to \`$net\`."
	fi

	slack_notify "#tradingcore-notify" ":scream:" "$msg"
	exit 1
}

success() {
	gitlab_ci="${GITLAB_CI:-false}"
	if test "$gitlab_ci" == "true" ; then
		commit_hash="${CI_COMMIT_SHORT_SHA:-[failed to get commit hash]}"
		commit_msg="${CI_COMMIT_TITLE:-[failed to get commit message]}"
		pipeline_url="${CI_PIPELINE_URL:-[failed to get pipeline URL]}"
		msg="\`$net\` has been deployed at \`$commit_hash\` \"$commit_msg\" (see $pipeline_url for details)."
	else
		msg="\`$net\` has been deployed."
	fi
	slack_notify "#engineering" ":tada:" "$msg"
}

test_ssh_access() {
	nodeloop "Testing ssh access" '/bin/true'
}

vega_resetconfig() {
	nodeloop \
		"Recreating vega config file" \
		'sudo -iu vega /bin/bash -c "' \
			'cd ; ./current/vega init -f' \
		'"'
}

# # #

# Specify which network is being deployed to
net="${1:-}"
if test -z "$net" ; then
	failure "Syntax: $0 somenet sshuser ..."
fi
net_ucase="$(echo "$net" | tr '[:lower:]' '[:upper:]')"

# Specify the user to ssh in to machines as
sshuser="${2:-}"
if test -z "$sshuser" ; then
	failure "Syntax: $0 somenet sshuser ..."
fi

shift
shift

check_vars

check_apps

ssh_setup

test_ssh_access

traders_action "stop"

stop_vega_tendermint

nukedata_tendermint
nukedata_vega

install_files "$@"

vega_resetconfig
start_vega_tendermint

traders_action "start"

ssh_tidy

success
exit 0
