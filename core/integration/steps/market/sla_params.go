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

package market

import (
	"embed"
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/steps/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/liquidity-sla-params/*.json
	defaultLiquiditySLAParams          embed.FS
	defaultLiquiditySLAParamsFileNames = []string{
		"defaults/liquidity-sla-params/default-basic.json",
		"defaults/liquidity-sla-params/default-futures.json",
		"defaults/liquidity-sla-params/default-st.json",
	}
)

type slaParams struct {
	config map[string]*types.LiquiditySLAParameters
}

func newLiquiditySLAParams(unmarshaler *defaults.Unmarshaler) *slaParams {
	liquiditySLAParams := &slaParams{
		config: map[string]*types.LiquiditySLAParameters{},
	}

	contentReaders := helpers.ReadAll(defaultLiquiditySLAParams, defaultLiquiditySLAParamsFileNames)
	for name, contentReader := range contentReaders {
		sla, err := unmarshaler.UnmarshalLiquiditySLAParams(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default SLA params %s: %v", name, err))
		}
		if err := liquiditySLAParams.Add(name, sla); err != nil {
			panic(fmt.Errorf("failed to add default liquidity SLA params %s: %v", name, err))
		}
	}

	return liquiditySLAParams
}

func (f *slaParams) Add(
	name string,
	config *types.LiquiditySLAParameters,
) error {
	f.config[name] = config
	return nil
}

func (f *slaParams) Get(name string) (*types.LiquiditySLAParameters, error) {
	config, ok := f.config[name]
	if !ok {
		return config, fmt.Errorf("no liquidity SLA params \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &types.LiquiditySLAParameters{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy liquidity SLA params: %v", err))
	}
	return copyConfig, nil
}
