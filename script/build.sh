#!/usr/bin/env bash

# Get a list of apps to build by looking at the directories in cmd.
mapfile -t apps < <(find cmd -maxdepth 1 -and -not -name cmd | sed -e 's#^cmd/##')

# Set a list of all targets.
alltargets=( \
	"linux/386" "linux/amd64" "linux/arm64" \
	"linux/mips" "linux/mipsle" "linux/mips64" "linux/mips64le" \
	"darwin/amd64" \
	"windows/386" "windows/amd64"
)

help() {
	echo "Command line arguments:"
	echo
	echo "  -a action  Take action: build, coverage, deps, install, integrationtest, test, race"
	echo "  -d         Build debug binaries"
	echo "  -T         Build all available GOOS+GOARCH combinations (see below)"
	echo "  -t list    Build specified GOOS+GOARCH combinations"
	echo "  -s suffix  Add arbitrary suffix to compiled binary names"
	echo "  -h         Show this help"
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

check_golang_version() {
	local goprog
	goprog="$(command -v go)"
	if test -z "$goprog" ; then
		echo "Could not find go"
		return 1
	fi

	goversion="$("$goprog" version)"
	if ! echo "$goversion" | grep -q 'go1\.14\.' ; then
		echo "Please use Go 1.14"
		echo "Using: $goprog"
		echo "Version: $goversion"
		return 1
	fi
}

deps() {
	mkdir -p "$GOPATH/pkg/mod/##@explicit" "$GOPATH/pkg/mod/@indirect" && \
	go mod download && \
	go mod vendor && \
	grep 'google/protobuf' go.mod | awk '{print "# " $1 " " $2 "\n"$1"/src";}' >> vendor/modules.txt && \
	grep 'tendermint/tendermint' go.mod | awk '{print "# " $1 " " $2 "\n"$1"/crypto/secp256k1/internal/secp256k1/libsecp256k1";}' >> vendor/modules.txt && \
	modvendor -copy="**/*.c **/*.h **/*.proto"
}


set_version() {
	version="${DRONE_TAG:-$(git describe --tags 2>/dev/null)}"
	version_hash="$(echo "${CI_COMMIT_SHA:-$(git rev-parse HEAD)}" | cut -b1-8)"
}

set_ldflags() {
	ldflags="-X main.Version=$version -X main.VersionHash=$version_hash"

	# The following ldflags are for running system-tests only - to shorten
	# durations to seconds/minutes instead of hours/days.
	if test -n "$VEGA_GOVERNANCE_MIN_CLOSE" ; then
		ldflags="$ldflags -X code.vegaprotocol.io/vega/governance.MinClose=$VEGA_GOVERNANCE_MIN_CLOSE"
	fi
	if test -n "$VEGA_GOVERNANCE_MAX_CLOSE" ; then
		ldflags="$ldflags -X code.vegaprotocol.io/vega/governance.MaxClose=$VEGA_GOVERNANCE_MAX_CLOSE"
	fi
	if test -n "$VEGA_GOVERNANCE_MIN_ENACT" ; then
		ldflags="$ldflags -X code.vegaprotocol.io/vega/governance.MinEnact=$VEGA_GOVERNANCE_MIN_ENACT"
	fi
	if test -n "$VEGA_GOVERNANCE_MAX_ENACT" ; then
		ldflags="$ldflags -X code.vegaprotocol.io/vega/governance.MaxEnact=$VEGA_GOVERNANCE_MAX_ENACT"
	fi
	if test -n "$VEGA_GOVERNANCE_MIN_PARTICIPATION_STAKE" ; then
		ldflags="$ldflags -X code.vegaprotocol.io/vega/governance.MinParticipationStake=$VEGA_GOVERNANCE_MIN_PARTICIPATION_STAKE"
	fi
}

parse_args() {
	# set defaults
	action=""
	gcflags=""
	dbgsuffix=""
	suffix=""
	targets=()

	while getopts 'a:ds:Tt:h' flag
	do
		case "$flag" in
		a)
			action="$OPTARG"
			;;
		d)
			gcflags="all=-N -l"
			dbgsuffix="-dbg"
			version="debug-$version"
			;;
		s)
			suffix="$OPTARG"
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
}

can_build() {
	local canbuild target
	canbuild=0
	target="$1" ; shift
	for compiler in "$@" ; do
		if test -z "$compiler" ; then
			continue
		fi

		if ! command -v "$compiler" 1>/dev/null ; then
			echo "$target: Cannot build. Need $compiler"
			canbuild=1
		fi
	done
	return "$canbuild"
}

set_go_flags() {
	local target
	target="$1" ; shift
	cc=""
	cgo_cflags="-I$PWD/vendor/github.com/tendermint/tendermint/crypto/secp256k1/internal/secp256k1 -I$PWD/vendor/github.com/tendermint/tendermint/crypto/secp256k1/internal/secp256k1/libsecp256k1"
	cgo_ldflags=""
	cgo_cxxflags=""
	cxx=""
	goarm=""
	if test "$target" == default ; then
		goarch=""
		goos=""
		osarchsuffix=""
	else
		goarch="$(echo "$target" | cut -f2 -d/)"
		goos="$(echo "$target" | cut -f1 -d/)"
		osarchsuffix="-$goos-$goarch"
	fi
	typesuffix=""
	skip=no
	if test "$action" == build ; then
		case "$target" in
		default)
			:
			;;
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
		windows/386)
			typesuffix=".exe"
			cc=i686-w64-mingw32-gcc-posix
			cxx=i686-w64-mingw32-g++-posix
			# https://docs.microsoft.com/en-us/cpp/porting/modifying-winver-and-win32-winnt?view=vs-2019
			win32_winnt="-D_WIN32_WINNT=0x0A00" # Windows 10
			cgo_cflags="$cgo_cflags $win32_winnt"
			cgo_cxxflags="$win32_winnt"
			;;
		windows/amd64)
			typesuffix=".exe"
			cc=x86_64-w64-mingw32-gcc-posix
			cxx=x86_64-w64-mingw32-g++-posix
			# https://docs.microsoft.com/en-us/cpp/porting/modifying-winver-and-win32-winnt?view=vs-2019
			win32_winnt="-D_WIN32_WINNT=0x0A00" # Windows 10
			cgo_cflags="$cgo_cflags $win32_winnt"
			cgo_cxxflags="$win32_winnt"
			;;
		*)
			echo "$target: Building this os+arch combination is TBD"
			skip=yes
			;;
		esac
	fi
	export \
		CC="$cc" \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$cgo_cflags" \
		CGO_LDFLAGS="$cgo_ldflags" \
		CGO_CXXFLAGS="$cgo_cxxflags" \
		CXX="$cxx" \
		GO111MODULE=on \
		GOARCH="$goarch" \
		GOARM="$goarm" \
		GOOS="$goos" \
		GOPROXY=direct \
		GOSUMDB=off

}

run() {
	check_golang_version
	parse_args "$@"
	if test -z "$action" ; then
		help
		exit 1
	fi
	if test "(" "$action" == build -o "$action" == install ")" -a -z "${targets[*]}" ; then
		help
		exit 1
	fi
	set_version
	set_ldflags
	echo "Version: $version ($version_hash)"

	set_go_flags default
	case "$action" in
	build)
		: # handled below
		;;
	coverage)
		c=.testCoverage.txt
		go list ./... | grep -v '/gateway' | xargs go test -covermode=count -coverprofile="$c" && \
			go tool cover -func="$c" && \
			go tool cover -html="$c" -o .testCoverage.html
		return $?
		;;
	deps)
		deps
		return "$?"
		;;
	install)
		: # handled below
		;;
	integrationtest)
		go test -v ./integration/... -godog.format=pretty
		return "$?"
		;;
	test)
		go test ./...
		return "$?"
		;;
	race)
		go test -race ./...
		return "$?"
		;;
	retest)
		go test -count=1 ./...
		return "$?"
		;;
	staticcheck)
		f="$(mktemp)"
		(
			go list ./... | grep -v /integration | xargs staticcheck
			find integration -name '*.go' -print0 | xargs -0 staticcheck | grep -v 'could not load export data'
		) | tee "$f"
		count="$(wc -l <"$f")"
		rm -f "$f"
		if test "$count" -gt 0 ; then
			return 1
		fi
		return 0
		;;
	*)
		echo "Invalid action: $action"
		return 1
	esac

	failed=0
	for target in "${targets[@]}" ; do
		set_go_flags "$target"
		if test "$skip" == yes ; then
			continue
		fi
		can_build "$target" "$cc" "$cxx" || continue

		log="/tmp/go.log"
		echo "$target: deps ... "
		deps 1>"$log" 2>&1
		code="$?"
		if test "$code" = 0 ; then
			echo "$target: deps OK"
		else
			echo "$target: deps failed ($code)"
			failed=$((failed+1))
			echo
			echo "=== BEGIN logs ==="
			cat "$log"
			echo "=== END logs ==="
			rm "$log"
			continue
		fi

		echo "$target: go get ... "
		go get -v . 1>"$log" 2>&1
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
			case "$action" in
			build)
				o="cmd/$app/$app$osarchsuffix$dbgsuffix$suffix$typesuffix"
				log="$o.log"
				msgprefix="$target: go $action $o ..."
				echo "$msgprefix"
				rm -f "$o" "$log"
				go build -v -ldflags "$ldflags" -gcflags "$gcflags" -o "$o" "./cmd/$app" 1>"$log" 2>&1
				code="$?"
				;;
			install)
				o="not/applicable"
				log="$app$osarchsuffix$dbgsuffix$suffix$typesuffix.log"
				msgprefix="$target: go $action $app ..."
				echo "$msgprefix"
				rm -f "$log"
				go install -v -ldflags "$ldflags" -gcflags "$gcflags" "./cmd/$app" 1>"$log" 2>&1
				code="$?"
				;;
			esac
			if test "$code" = 0 ; then
				echo "$msgprefix OK"
				rm "$log"
			else
				echo "$msgprefix failed ($code)"
				failed=$((failed+1))
				echo
				echo "=== BEGIN logs for $msgprefix ==="
				cat "$log"
				echo "=== END logs for $msgprefix ==="
			fi
		done
	done
	if test "$failed" -gt 0 ; then
		echo "Build failed for $failed apps."
	fi
	return "$failed"
}

# # #

if echo "$0" | grep -q '/build.sh$' ; then
	# being run as a script
	run "$@"
	exit "$?"
fi
