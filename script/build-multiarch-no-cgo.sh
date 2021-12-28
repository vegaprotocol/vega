#!/usr/bin/env bash

# Get a list of apps to build by looking at the directories in cmd.
mapfile -t apps < <(find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##' | sort)

default_combinations=(
	linux-amd64
	darwin-amd64
	windows-amd64
)

all_combinations=("${default_combinations[@]}")
all_combinations+=(
	linux-386
	linux-arm64
	linux-mips64
	linux-mips64le
	linux-mips
	linux-mipsle
	windows-386
)

case "${1:-}" in
	"")
		combinations=("${default_combinations[@]}")
		;;
	default)
		combinations=("${default_combinations[@]}")
		;;
	all)
		combinations=("${all_combinations[@]}")
		;;
	*)
		echo "Invalid: $1"
		exit 1
		;;
esac

build_app() {
	local app
	app="${1:?}"

	local combo goarch goos ldflags o pidsfile

	case "$app" in
		vega)
			version="${DRONE_TAG:-$(git describe --tags 2>/dev/null)}"
			version_hash="$(echo "${CI_COMMIT_SHA:-$(git rev-parse HEAD)}" | cut -b1-8)"
			ldflags="-X main.CLIVersion=$version -X main.CLIVersionHash=$version_hash"
			;;
		*)
			ldflags=""
			;;
	esac

	pidsfile="$(mktemp)"
	for combo in "${combinations[@]}" ; do
		goos="$(echo "$combo" | cut -f1 -d-)"
		goarch="$(echo "$combo" | cut -f2 -d-)"
		case "$goos" in
			windows) o="cmd/$app/$app-$goos-$goarch.exe" ;;
			*) o="cmd/$app/$app-$goos-$goarch" ;;
		esac
		env \
			CGO_ENABLED=0 \
			GOOS="$goos" \
			GOARCH="$goarch" \
			go build -o "$o" -ldflags "$ldflags" "./cmd/$app" &
		pid="$!"
		echo "Building $app for OS=$goos arch=$goarch in subprocess $pid"
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

	if test "$final" -gt 0 ; then
		echo "Failed to build $app"
		exit "$final"
	fi
}

for app in "${apps[@]}" ; do
	build_app "$app"
done
