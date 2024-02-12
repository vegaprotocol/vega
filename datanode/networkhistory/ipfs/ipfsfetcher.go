// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	kuboClient "github.com/ipfs/kubo/client/rpc"
	"github.com/ipfs/kubo/repo/fsrepo/migrations"
)

const (
	shellUpTimeout    = 2 * time.Second
	defaultFetchLimit = 1024 * 1024 * 512
)

type ipfsFetcher struct {
	distPath string
	ipfsDir  string
	limit    int64
}

// newIpfsFetcher creates a new IpfsFetcher
//
// Specifying "" for distPath sets the default IPNS path.
// Specifying 0 for fetchLimit sets the default, -1 means no limit.
func newIpfsFetcher(distPath string, ipfsDir string, fetchLimit int64) *ipfsFetcher {
	f := &ipfsFetcher{
		limit:    defaultFetchLimit,
		distPath: migrations.LatestIpfsDist,
		ipfsDir:  ipfsDir,
	}

	if distPath != "" {
		if !strings.HasPrefix(distPath, "/") {
			distPath = "/" + distPath
		}
		f.distPath = distPath
	}

	if fetchLimit != 0 {
		if fetchLimit == -1 {
			fetchLimit = 0
		}
		f.limit = fetchLimit
	}

	return f
}

func (f *ipfsFetcher) Close() error {
	return nil
}

// Fetch attempts to fetch the file at the given path, from the distribution
// site configured for this HttpFetcher.  Returns io.ReadCloser on success,
// which caller must close.
func (f *ipfsFetcher) Fetch(ctx context.Context, filePath string) ([]byte, error) {
	sh, err := kuboClient.NewPathApi(f.ipfsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create a ipfs shell migration: %w", err)
	}
	resp, err := sh.Request("cat", path.Join(f.distPath, filePath)).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from the ipfs node: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	defer resp.Close()

	var output io.Reader
	if f.limit != 0 {
		output = migrations.NewLimitReadCloser(resp.Output, f.limit)
	} else {
		output = resp.Output
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(output)

	return buf.Bytes(), nil
}
