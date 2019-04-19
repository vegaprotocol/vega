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

check_apps() {
	# Check required programs
	apps=( rsync scp ssh )
	for app in "${apps[@]}" ; do
		if ! which "$app" 1>/dev/null ; then
			echo "Program missing: $app" >/dev/stderr
			exit 1
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
				echo "Variable missing: \$${var}" >/dev/stderr
				exit 1
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
		echo "Failed to run commands on all nodes. Max exit code was $maxcode" >/dev/stderr
		exit 1
	fi
}

nukedata_tendermint() {
	nodeloop \
		"Resetting tendermint chain" \
		'sudo -iu vega /bin/bash -c "' \
			'cd ; ./tendermint unsafe_reset_all' \
		'"'
}

nukedata_vega() {
	nodeloop \
		"Deleting vega stores" \
		'sudo -iu vega /bin/bash -c "' \
			'cd ; rm -rf current/tmp/*store .vega/*store' \
		'"'
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
		'./tendermint version ; ' \
		'sudo systemctl daemon-reload ; ' \
		'sudo systemctl restart vega ; ' \
		'sleep 1 ; ' \
		'sudo systemctl restart tendermint'
}

stop_vega_tendermint() {
	nodeloop \
		"Stopping vega and tendermint processes with SystemD" \
		'cd ; ./current/vega --version ; ' \
		'./tendermint version ; ' \
		'sudo systemctl daemon-reload ; ' \
		'sudo systemctl stop vega ; ' \
		'sudo systemctl stop tendermint ; ' \
		'sudo killall vega tendermint 1>/dev/null 2>&1 ; ' \
		'true'
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
	echo "Syntax: $0 somenet sshuser ..." >/dev/stderr
	exit 1
fi
net_ucase="$(echo "$net" | tr '[:lower:]' '[:upper:]')"

# Specify the user to ssh in to machines as
sshuser="${2:-}"
if test -z "$sshuser" ; then
	echo "Syntax: $0 somenet sshuser ..." >/dev/stderr
	exit 1
fi

shift
shift

check_vars

check_apps

ssh_setup

test_ssh_access

stop_vega_tendermint

nukedata_tendermint
nukedata_vega

install_files "$@"

vega_resetconfig
start_vega_tendermint

ssh_tidy

exit 0
