package main

import (
	"code.vegaprotocol.io/vega/basecmd"
	"code.vegaprotocol.io/vega/basecmd/auth"
	"code.vegaprotocol.io/vega/basecmd/gateway"
	"code.vegaprotocol.io/vega/basecmd/initcmd"
	"code.vegaprotocol.io/vega/basecmd/node"
	"code.vegaprotocol.io/vega/basecmd/version"

	_ "code.vegaprotocol.io/vega/plugins/orders"
)

const (
	defaultVersionHash = "unknown"
	defaultVersion     = "unknown"
)

var (
	// VersionHash specifies the git commit used to build the application. See VERSION_HASH in Makefile for details.
	VersionHash = ""

	// Version specifies the version used to build the application. See VERSION in Makefile for details.
	Version = ""
)

func main() {
	if len(VersionHash) <= 0 {
		VersionHash = defaultVersionHash
	}
	if len(Version) <= 0 {
		Version = defaultVersion
	}

	basecmd.Version = Version
	basecmd.VersionHash = VersionHash

	basecmd.Main(
		auth.Command,
		gateway.Command,
		initcmd.Command,
		node.Command,
		version.Command,
	)
}
