#!/bin/bash

mapfile -t apps < <(find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##')
alltargets=("linux/amd64" "linux/386" "darwin/amd64")

help() {
	echo "Command line arguments:"
	echo
	echo "  -d       Build debug binaries"
	echo "  -T       Build all available GOOS+GOARCH combinations (see below)"
	echo "  -t list  Build specified GOOS+GOARCH combinations"
	echo "  -h       Show this help"
	echo
	echo "Apps to be built:"
	for app in "${apps[@]}" ; do
		echo "  * $app"
	done
	echo
	echo "Available targets:"
	for target in "${alltargets[@]}" ; do
		echo "  * $target"
	done
}

set_version() {
	if test -n "${DRONE:-}" ; then
		# In Drone CI
		version="${DRONE_TAG:-$(git describe --tags 2>/dev/null)}"
		version_hash="$(echo "${CI_COMMIT_SHA:-nohash}" | cut -b1-8)"
		return
	fi
	version="dev-${USER:-unknownuser}"
	version_hash="$(git rev-parse HEAD | cut -b1-8)"
}

parse_args() {
	# set defaults
	gcflags=""
	suffix=""
	targets=()

	while getopts 'dTt:h' flag; do
		case "$flag" in
		d)
			gcflags="all=-N -l"
			suffix="-dbg"
			version="debug-$version"
			;;
		t)
			mapfile -t targets < <(echo "$OPTARG" | tr ' ,' '\n')
			;;
		T)
			targets=("${alltargets[@]}")
			;;
		h)
			help
			exit 0
			;;
		*)
			echo "Invalid option: $flag"
			exit 1
			;;
		esac
	done
	ldflags="-X main.Version=$version -X main.VersionHash=$version_hash"
	if test -z "${targets[*]}" ; then
		help
	else
		echo "Version: $version ($version_hash)"
	fi
}

run() {
	set_version
	parse_args "$@"

	failed=0
	for target in "${targets[@]}" ; do
		goos="$(echo "$target" | cut -f1 -d/)"
		goarch="$(echo "$target" | cut -f2 -d/)"
		for app in "${apps[@]}" ; do
			o="cmd/$app/$app-$goos-$goarch$suffix" ; \
			log="$o.log"
			echo -n "Building $o ... "
			case "$target" in
			darwin/*)
				env \
					CC=o64-clang CXX=o64-clang++ \
					GOOS="$goos" GOARCH="$goarch" \
					CGO_ENABLED=1 \
					go build -v \
					-ldflags "$ldflags" \
					-gcflags "$gcflags" \
					-o "$o" "./cmd/$app" \
					1>"$log" 2>&1
				code="$?"
				;;
			linux/*)
				env \
					GOOS="$goos" GOARCH="$goarch" \
					CGO_ENABLED=1 \
					go build -v \
					-ldflags "$ldflags" \
					-gcflags "$gcflags" \
					-o "$o" "./cmd/$app" \
					1>"$log" 2>&1
				code="$?"
				;;
			*)
				echo "TBD" | tee "$log"
				code=1
				;;
			esac
			if test "$code" = 0 ; then
				echo "OK"
			else
				echo "Exit code $code" >>"$log"
				echo "failed"
				failed=$((failed+1))
				echo
				echo "=== BEGIN logs for $o ==="
				cat "$log"
				echo "=== END logs for $o ==="
			fi
			rm "$log"
		done
	done
	return "$failed"
}

# # #

if echo "$0" | grep -q '/build.sh$' ; then
	# being run as a script
	run "$@"
	failed="$?"
	if test "$failed" -gt 0 ; then
		echo "Build failed for $failed apps."
	fi
	exit "$?"
fi
