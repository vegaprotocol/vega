#!/usr/bin/env bash

default_combinations="linux,amd64
	darwin,amd64
	windows,amd64"

all_combinations="$default_combinations
	linux,386
	linux,arm64
	linux,mips64
	linux,mips64le
	linux,mips
	linux,mipsle
	windows,386"

case "${1:-}" in
	"") combinations="$default_combinations" ;;
	default) combinations="$default_combinations" ;;
	all) combinations="$all_combinations" ;;
	*) echo "Invalid: $1" ; exit 1 ;;
esac

pidsfile="$(mktemp)"
for combo in $combinations ; do
	goos="$(echo "$combo" | cut -f1 -d,)"
	goarch="$(echo "$combo" | cut -f2 -d,)"
	case "$goos" in
		windows) o="cmd/vega/vega-$goos-$goarch.exe" ;;
		*) o="cmd/vega/vega-$goos-$goarch" ;;
	esac
	env \
		CGO_ENABLED=0 \
		GOOS="$goos" \
		GOARCH="$goarch" \
		go build -o "$o" ./cmd/vega &
	pid="$!"
	echo "Building for OS=$goos arch=$goarch in subprocess $pid"
	echo "$pid" >>"$pidsfile"
done

final=0
while read -r pid ; do
	echo -n "Waiting for subprocess $pid ..."
	wait "$pid"
	code="$?"
	if test "$code" = 0 ; then
		echo "OK"
	else
		echo "code $code"
		final="$((final+1))"
	fi
done <"$pidsfile"
rm -f "$pidsfile"

exit "$final"
