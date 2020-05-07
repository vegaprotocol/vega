#!/bin/bash

# Get a list of apps to build by looking at the directories in cmd.
mapfile -t apps < <(find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##')

# Set a list of all targets.
alltargets=( \
	"linux/386" "linux/amd64" "linux/arm64" \
	"linux/mips" "linux/mipsle" "linux/mips64" "linux/mips64le" \
	"darwin/amd64" \
)

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
		cc=""
		cgo_enabled=1
		cgo_flags=""
		cgo_ldflags=""
		cgo_cxxflags=""
		cxx=""
		goarm=""
		goarch="$(echo "$target" | cut -f2 -d/)"
		goos="$(echo "$target" | cut -f1 -d/)"
		skip=no
		case "$target" in
		darwin/*)
			cc=o64-clang
			cxx=o64-clang++
			;;
		linux/386)
			:
			;;
		linux/amd64)
			:
			;;
		linux/arm64)
			cc=aarch64-linux-gnu-gcc-9
			cxx=aarch64-linux-gnu-g++-9
			;;
		linux/mips)
			cc=mips-linux-gnu-gcc-9
			cxx=mips-linux-gnu-g++-9
			;;
		linux/mipsle)
			cc=mipsel-linux-gnu-gcc-9
			cxx=mipsel-linux-gnu-g++-9
			;;
		linux/mips64)
			cc=mips64-linux-gnuabi64-gcc-9
			cxx=mips64-linux-gnuabi64-g++-9
			;;
		linux/mips64le)
			cc=mips64el-linux-gnuabi64-gcc-9
			cxx=mips64el-linux-gnuabi64-g++-9
			;;
		windows/*)
			o="$o.exe"
			cc=x86_64-w64-mingw32-gcc-posix
			cxx=x86_64-w64-mingw32-g++-posix
			# https://docs.microsoft.com/en-us/cpp/porting/modifying-winver-and-win32-winnt?view=vs-2019
			win32_winnt="-D_WIN32_WINNT=0x0A00" # Windows 10
			cgo_flags="$win32_winnt"
			cgo_cxxflags="$win32_winnt"
			;;
		*)
			echo "$target: Building this os+arch combination is TBD"
			skip=yes
			;;
		esac

		if test "$skip" == yes ; then
			continue
		fi

		export \
			CC="$cc" \
			CGO_ENABLED="$cgo_enabled" \
			CGO_CFLAGS="$cgo_flags" \
			CGO_LDFLAGS="$cgo_ldflags" \
			CGO_CXXFLAGS="$cgo_cxxflags" \
			CXX="$cxx" \
			GOARCH="$goarch" \
			GOARM="$goarm" \
			GOOS="$goos"

		log="/tmp/go.log"
		echo "$target: go mod download ... "
		go mod download 1>"$log" 2>&1
		code="$?"
		if test "$code" = 0 ; then
			echo "$target: go mod download OK"
		else
			echo "$target: go mod download failed ($code)"
			failed=$((failed+1))
			echo
			echo "=== BEGIN logs ==="
			cat "$log"
			echo "=== END logs ==="
			rm "$log"
			continue
		fi

		echo "$target: go mod vendor ... "
		go mod vendor 1>"$log" 2>&1
		code="$?"
		if test "$code" = 0 ; then
			echo "$target: go mod vendor OK"
		else
			echo "$target: go mod vendor failed ($code)"
			failed=$((failed+1))
			echo
			echo "=== BEGIN logs ==="
			cat "$log"
			echo "=== END logs ==="
			rm "$log"
			continue
		fi

		grep 'google/protobuf' go.mod | awk '{print "# " $$1 " " $$2 "\n"$1"/src";}' >>vendor/modules.txt
		mkdir -p "$GOPATH/pkg/mod/@indirect"

		echo "$target: modvendor ... "
		modvendor -copy="**/*.proto" 1>"$log" 2>&1
		code="$?"
		if test "$code" = 0 ; then
			echo "$target: modvendor OK"
		else
			echo "$target: modvendor failed ($code)"
			failed=$((failed+1))
			echo
			echo "=== BEGIN logs ==="
			cat "$log"
			echo "=== END logs ==="
			rm "$log"
			continue
		fi

		echo "$target: go get ... "
		go get -v -ldflags "$ldflags" . 1>"$log" 2>&1
		code="$?"
		if test "$code" = 0 ; then
			echo "$target: go get OK"
		else
			echo "$target: go get failed ($code)"
			failed=$((failed+1))
			echo
			echo "=== BEGIN logs ==="
			cat "$log"
			echo "=== END logs ==="
			rm "$log"
			continue
		fi

		for app in "${apps[@]}" ; do
			o="cmd/$app/$app-$goos-$goarch$suffix"
			log="$o.log"
			echo "$target: go build $o ... "
			rm -f "$o" "$log"
			go build -v -ldflags "$ldflags" -gcflags "$gcflags" -o "$o" "./cmd/$app" 1>"$log" 2>&1
			code="$?"
			if test "$code" = 0 ; then
				echo "$target: go build $o OK"
				rm "$log"
			else
				echo "$target: go build $o failed ($code)"
				failed=$((failed+1))
				echo
				echo "=== BEGIN logs for $o ==="
				cat "$log"
				echo "=== END logs for $o ==="
			fi
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
