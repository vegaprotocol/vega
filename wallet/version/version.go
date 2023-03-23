package version

import (
	"runtime/debug"

	vgversion "code.vegaprotocol.io/shared/libs/version"

	"github.com/blang/semver/v4"
)

const (
	ReleasesAPI = "https://api.github.com/repos/vegaprotocol/vega/releases"
	ReleasesURL = "https://github.com/vegaprotocol/vega/releases"
)

var (
	// GitHash specifies the git commit used to build the application.
	GitHash = "unknown"

	// Version specifies the version used to build the application.
	Version = "v0.53.2"
)

func IsUnreleased() bool {
	return vgversion.IsUnreleased(Version)
}

type GetVersionResponse struct {
	Version string `json:"version"`
	GitHash string `json:"gitHash"`
}

func GetVersionInfo() *GetVersionResponse {
	return &GetVersionResponse{
		Version: Version,
		GitHash: GitHash,
	}
}

func Check(releasesGetterFn vgversion.ReleasesGetter, currentRelease string) (*semver.Version, error) {
	return vgversion.Check(releasesGetterFn, currentRelease)
}

func init() {
	info, _ := debug.ReadBuildInfo()
	modified := false

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			GitHash = v.Value
		}
		if v.Key == "vcs.modified" {
			modified = true
		}
	}
	if modified {
		GitHash += "-modified"
	}
}
