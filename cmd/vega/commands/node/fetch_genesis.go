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

package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"code.vegaprotocol.io/vega/core/genesis"
	tmtypes "github.com/tendermint/tendermint/types"
)

func genesisDocHTTPFromURL(genesisFilePath string) (*tmtypes.GenesisDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
