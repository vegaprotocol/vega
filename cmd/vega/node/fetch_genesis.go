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

package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"code.vegaprotocol.io/vega/core/genesis"
	tmtypes "github.com/cometbft/cometbft/types"
)

func genesisDocHTTPFromURL(genesisFilePath string) (*tmtypes.GenesisDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, genesisFilePath, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't load genesis file from %s: %w", genesisFilePath, err)
	}
	defer resp.Body.Close()
	jsonGenesis, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, _, err := genesis.FromJSON(jsonGenesis)
	if err != nil {
		return nil, fmt.Errorf("invalid genesis file from %s: %w", genesisFilePath, err)
	}

	return doc, nil
}

func httpGenesisDocProvider(networkSelect string) (*tmtypes.GenesisDoc, error) {
	genesisFilesRootPath := fmt.Sprintf("https://raw.githubusercontent.com/vegaprotocol/networks/master/%s", networkSelect)

	doc, _, err := getGenesisFromRemote(genesisFilesRootPath)

	return doc, err
}

func getGenesisFromRemote(genesisFilesRootPath string) (*tmtypes.GenesisDoc, *genesis.State, error) {
	jsonGenesis, err := fetchData(fmt.Sprintf("%s/genesis.json", genesisFilesRootPath))
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get remote genesis file: %w", err)
	}
	doc, state, err := genesis.FromJSON(jsonGenesis)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't parse genesis file: %w", err)
	}
	return doc, state, nil
}

func fetchData(path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't build request for %s: %w", path, err)
	}
	sigResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't get response for %s: %w", path, err)
	}
	defer sigResp.Body.Close()
	data, err := ioutil.ReadAll(sigResp.Body)
	if err != nil {
		return nil, fmt.Errorf("couldn't read response body: %w", err)
	}
	return data, nil
}
