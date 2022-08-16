package version

import (
	vgversion "code.vegaprotocol.io/vega/libs/version"
	coreversion "code.vegaprotocol.io/vega/version"

	"github.com/blang/semver/v4"
)

const (
	ReleasesAPI = "https://api.github.com/repos/vegaprotocol/vega/releases"
	ReleasesURL = "https://github.com/vegaprotocol/vega/releases"
)

func IsUnreleased() bool {
	return vgversion.IsUnreleased(coreversion.Get())
}

type GetVersionResponse struct {
	Version string `json:"version"`
	GitHash string `json:"gitHash"`
}

func GetVersionInfo() *GetVersionResponse {
	return &GetVersionResponse{
		Version: coreversion.Get(),
		GitHash: coreversion.GetCommitHash(),
	}
}

func Check(releasesGetterFn vgversion.ReleasesGetter, currentRelease string) (*semver.Version, error) {
	return vgversion.Check(releasesGetterFn, currentRelease)
}
