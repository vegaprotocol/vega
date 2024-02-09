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
	//go:embed defaults/liquidation-config/*.json
	defaultLiquidationConfigs         embed.FS
	defaultLiquidationConfigFileNames = []string{
		"defaults/liquidation-config/legacy-liquidation-strategy.json",
		"defaults/liquidation-config/default-liquidation-strat.json",
		"defaults/liquidation-config/slow-liquidation-strat.json",
		"defaults/liquidation-config/AC-013-strat.json",
	}

	defaultStrat = "legacy-liquidation-strategy"
)

type liquidationConfig struct {
	config map[string]*types.LiquidationStrategy
}

func newLiquidationConfig(unmarshaler *defaults.Unmarshaler) *liquidationConfig {
	config := &liquidationConfig{
		config: map[string]*types.LiquidationStrategy{},
	}

	contentReaders := helpers.ReadAll(defaultLiquidationConfigs, defaultLiquidationConfigFileNames)
	for name, contentReader := range contentReaders {
		liquidationConfig, err := unmarshaler.UnmarshalLiquidationConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default liquidation config %s: %v", name, err))
		}
		if err := config.Add(name, liquidationConfig); err != nil {
			panic(fmt.Errorf("failed to add default liquidation config %s: %v", name, err))
		}
	}

	return config
}

func (f *liquidationConfig) Add(name string, strategy *types.LiquidationStrategy) error {
	f.config[name] = strategy
	return nil
}

func (f *liquidationConfig) Get(name string) (*types.LiquidationStrategy, error) {
	if name == "" {
		// for now, default to the legacy strategy
		name = defaultStrat
	}
	strategy, ok := f.config[name]
	if !ok {
		return strategy, fmt.Errorf("no liquidation configuration \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &types.LiquidationStrategy{}
	if err := copier.Copy(copyConfig, strategy); err != nil {
		panic(fmt.Errorf("failed to deep copy liquidation config: %v", err))
	}
	return copyConfig, nil
}
