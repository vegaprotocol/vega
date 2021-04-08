#!/usr/bin/env bash

pidsfile="$(mktemp)"
for combo in \
	linux,amd64 \
	linux,386 \
	linux,arm64 \
	darwin,amd64 \
	linux,mips64 \
	linux,mips64le \
	linux,mips \
	linux,mipsle \
	windows,amd64 \
	windows,386
do
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
