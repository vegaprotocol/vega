// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package market

import (
	"embed"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/protos/vega"
)

var (
	//go:embed defaults/liquidity-sla-params/*.json
	defaultLiquiditySLAParams          embed.FS
	defaultLiquiditySLAParamsFileNames = []string{
		"defaults/liquidity-sla-params/default-basic.json",
		"defaults/liquidity-sla-params/default-futures.json",
	}
)

type slaParams struct {
	config map[string]*types.LiquiditySLAParameters
}

func newLiquiditySLAParams(unmarshaler *defaults.Unmarshaler) *slaParams {
	liquiditySLAParams := &slaParams{
		config: map[string]*types.LiquiditySLAParameters{},
	}

	contentReaders := defaults.ReadAll(defaultLiquiditySLAParams, defaultLiquiditySLAParamsFileNames)
	for name, contentReader := range contentReaders {
		sla, err := unmarshaler.UnmarshalLliquiditySLAParams(contentReader)
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
