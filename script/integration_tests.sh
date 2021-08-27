#!/usr/bin/env bash

# This script runs individual integration tests in parallel, and prints the
# feature file name for any failing tests.

# The script will run tests on the feature files specified on the commandline.
# If none are given, it will run tests using ALL feature files.

# Set the following env vars, or leave the defaults
jobs="${JOBS:-$(grep -c ^processor /proc/cpuinfo 2>/dev/null || echo 4)}"
verbose="${VERBOSE:-no}"

if [[ "$#" = 0 ]] ; then
	[[ "$verbose" = yes ]] && echo "No filename arguments given. Finding all feature files and running test in parallel (batches of $jobs)."
	(cd integration/ && find features -name '*.feature') | xargs -n"$jobs" /usr/bin/env bash "$0"
	exit "$?"
fi

green=""
red=""
nocol=""
if test -t 1 ; then
	green="\033[36m"
	red="\033[31m"
	nocol="\033[0m"
fi

one_test() {
	local featurefile
	featurefile="${1:?}"

	local outcome
	outcome="$(go test -v ./integration/ --godog.format=progress "$featurefile" | tail -1 | cut -f1 -d ' ')"
	if [[ "$outcome" = ok ]] ; then
		[[ "$verbose" = yes ]] && echo -e "${green}OK  ${nocol}: $featurefile"
		return 0
	fi
	echo -e "${red}FAIL${nocol}: $featurefile"
	return 1
}

[[ "$verbose" = yes ]] && echo "Running $# tests in parallel."
pids=()
for f in "$@" ; do
	one_test "$f" &
	pid="$!"
	[[ "$verbose" = yes ]] && echo "Subprocess $pid for $f"
	pids+=("$pid")
done

[[ "$verbose" = yes ]] && echo "Waiting for $# tests to finish."
lastcode=0
for pid in "${pids[@]}" ; do
	[[ "$verbose" = yes ]] && echo "Waiting for subprocess $pid ..."
	wait "$pid"
	code="$?"
	[[ "$verbose" = yes ]] && echo "Subprocess $pid finished with code $code."
	if [[ "$code" != 0 ]] ; then
		lastcode=1
	fi
done
exit "$lastcode"
