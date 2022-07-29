package version

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
)

// ReleasesGetter return the list of releases as semantic version strings
type ReleasesGetter func() ([]*Version, error)

// IsUnreleased tells if the version in parameter is an unreleased version or
// not. An unreleased version is a development version from the semantic
// versioning point of view.
// This doesn't probe GitHub for its metadata on the version, and this is
// intended. A release flagged as a pre-release in GitHub is just to mark mainnet
// incompatibility.
func IsUnreleased(version string) bool {
	v, err := NewVersionFromString(version)
	if err != nil {
		// unsupported version, considered unreleased
		return true
	}

	return !v.IsReleased
}

// Check returns a newer version, or an error or nil for both if no error
// happened, and no updates are needed.
func Check(releasesGetterFn ReleasesGetter, currentRelease string) (*semver.Version, error) {
	currentVersion, err := NewVersionFromString(currentRelease)
	if err != nil {
		return nil, fmt.Errorf("couldn't extract version from release: %w", err)
	}
	latestVersion := currentVersion

	releases, err := releasesGetterFn()
	if err != nil {
		return nil, fmt.Errorf("couldn't get releases: %w", err)
	}

	var updateAvailable bool
	for _, newVersion := range releases {
		if shouldUpdate(latestVersion, newVersion) {
			updateAvailable = true
			latestVersion = newVersion
		}
	}

	if !updateAvailable {
		return nil, nil
	}

	return latestVersion.Version, nil
}

func shouldUpdate(latestVersion *Version, newVersion *Version) bool {
	if newVersion.IsDraft {
		return false
	}

	if latestVersion.IsReleased && !newVersion.IsReleased {
		return false
	}

	if latestVersion.IsDevelopment && nonDevelopmentVersionAvailable(latestVersion, newVersion) {
		return true
	}

	return newVersion.Version.GT(*latestVersion.Version)
}

// nonDevelopmentVersionAvailable verifies if the compared version is the
// non-development equivalent of the latest version.
// For example, 0.9.0-pre1 is the non-development version of 0.9.0-pre1+dev.
// In semantic versioning, we don't compare the `build` annotation, so verifying
// equality between 0.9.0-pre1 and 0.9.0-pre1+dev results in comparing:
//     0.9.0-pre1 <> 0.9.0-pre1
// So if it's equal, it means we have a
func nonDevelopmentVersionAvailable(latestVersion *Version, comparedVersion *Version) bool {
	return comparedVersion.Version.EQ(*latestVersion.Version)
}

// NewVersionFromString creates a Version and set the appropriate flags on it
// based on the segments that compose the version.
func NewVersionFromString(release string) (*Version, error) {
	v, err := semver.New(strings.TrimPrefix(release, "v"))
	if err != nil {
		return nil, err
	}

	version := &Version{
		Version: v,
	}

	for _, build := range v.Build {
		if build == "dev" {
			version.IsDevelopment = true
		}
	}

	version.IsPreReleased = len(v.Pre) != 0

	version.IsReleased = !version.IsDevelopment && !version.IsPreReleased && !version.IsDraft

	return version, nil
}

type Version struct {
	Version       *semver.Version
	IsDraft       bool
	IsDevelopment bool
	IsPreReleased bool
	IsReleased    bool
}
