package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/blang/semver/v4"
)

type githubReleaseResponse struct {
	Name         string `json:"name"`
	IsDraft      bool   `json:"draft"`
	IsPreRelease bool   `json:"prerelease"`
}

func GetGithubReleaseURL(releasesURL string, v *semver.Version) string {
	return fmt.Sprintf("%v/tag/v%v", releasesURL, v)
}

func BuildGithubReleasesRequestFrom(ctx context.Context, releasesURL string) ReleasesGetter {
	return func() ([]*Version, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
		if err != nil {
			return nil, fmt.Errorf("couldn't build request: %w", err)
		}
		req.Header.Add("Accept", "application/vnd.github.v3+json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("couldn't deliver request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("couldn't read response body: %w", err)
		}

		responses := []githubReleaseResponse{}
		if err = json.Unmarshal(body, &responses); err != nil {
			// try to parse as a general error message which would be useful information
			// to know e.g. if we were blocked due to GitHub rate-limiting
			m := struct {
				Message string `json:"message"`
			}{}
			if mErr := json.Unmarshal(body, &m); mErr == nil {
				return nil, fmt.Errorf("couldn't read response message: %s: %w", m.Message, err)
			}

			return nil, fmt.Errorf("couldn't unmarshal response body: %w", err)
		}

		releases := []*Version{}
		for _, response := range responses {
			release, err := NewVersionFromString(response.Name)
			if err != nil {
				// unsupported version
				continue
			}

			// At this point, the Version has been initialised based on the
			// segment of the version string. We have to readjust it based on
			// GitHub metadata.

			// We set the draft flag from GitHub response unconditionally as this
			// can only be inferred from GitHub metadata.
			release.IsDraft = response.IsDraft

			// If this is not marked as pre-release already, this means it's
			// either a stable version, either a temporary pre-release.
			// Temporary pre-release are releases that are supposed to be genuine
			// ones, but have been temporarily marked as pre-release in GitHub
			// to warn incompatibility with mainnet.
			// As a result, if not set, we verify if the state in GitHub.
			if !release.IsPreReleased {
				release.IsPreReleased = response.IsPreRelease
			}

			// We recompute the release flag based on the update above.
			release.IsReleased = !release.IsDevelopment && !release.IsPreReleased && !release.IsDraft

			releases = append(releases, release)
		}

		return releases, nil
	}
}
