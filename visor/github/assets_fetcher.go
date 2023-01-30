// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"code.vegaprotocol.io/vega/visor/utils"
	"github.com/google/go-github/v50/github"
	"golang.org/x/sync/errgroup"
)

type AssetsFetcher struct {
	repositoryOwner string
	repository      string

	assetNames map[string]struct{}

	*github.Client
}

func NewAssetsFetcher(
	repositoryOwner, repository string,
	assetsNames []string,
) *AssetsFetcher {
	return &AssetsFetcher{
		repositoryOwner: repositoryOwner,
		repository:      repository,
		assetNames:      utils.ToLookupMap(assetsNames),
		Client:          github.NewClient(nil),
	}
}

func (af *AssetsFetcher) GetReleaseID(ctx context.Context, releaseTag string) (int64, error) {
	releases, _, err := af.Client.Repositories.ListReleases(ctx, af.repositoryOwner, af.repository, nil)
	if err != nil {
		return 0, err
	}

	for _, r := range releases {
		if *r.TagName == releaseTag {
			return r.GetID(), nil
		}
	}

	return 0, fmt.Errorf("release tag %q not found", releaseTag)
}

func (af *AssetsFetcher) GetAssets(ctx context.Context, releaseID int64) ([]*github.ReleaseAsset, error) {
	assets, _, err := af.Client.Repositories.ListReleaseAssets(ctx, af.repositoryOwner, af.repository, releaseID, nil)
	if err != nil {
		return nil, err
	}

	var filteredAssets []*github.ReleaseAsset
	for _, asset := range assets {
		if _, ok := af.assetNames[asset.GetName()]; ok {
			filteredAssets = append(filteredAssets, asset)
		}
	}

	return filteredAssets, nil
}

func (af *AssetsFetcher) DownloadAsset(ctx context.Context, assetID int64, path string) error {
	followClient := &http.Client{Timeout: time.Second * 120}

	ra, _, err := af.Client.Repositories.DownloadReleaseAsset(ctx, af.repositoryOwner, af.repository, assetID, followClient)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	defer ra.Close()

	all, err := ioutil.ReadAll(ra)
	if err != nil {
		return fmt.Errorf("failed to read  %q: %w", path, err)
	}

	if err := os.WriteFile(path, all, 0o770); err != nil {
		return fmt.Errorf("failed to write to %q: %w", path, err)
	}

	return nil
}

func (af *AssetsFetcher) Download(ctx context.Context, releaseTag, downloadDir string) error {
	releaseID, err := af.GetReleaseID(ctx, releaseTag)
	if err != nil {
		return fmt.Errorf("failed to get release ID for tag %q: %q", releaseTag, err)
	}

	assetIDs, err := af.GetAssets(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("failed to get assets ID for tag %q: %q", releaseTag, err)
	}

	eg, ctx := errgroup.WithContext(ctx)
	for _, asset := range assetIDs {
		assetID := asset.GetID()
		assetName := asset.GetName()

		eg.Go(func() error {
			if err := af.DownloadAsset(ctx, assetID, path.Join(downloadDir, assetName)); err != nil {
				return fmt.Errorf("failed to download asset %q for tag %q: %w", assetName, releaseTag, err)
			}
			return nil
		})
	}

	return eg.Wait()
}
