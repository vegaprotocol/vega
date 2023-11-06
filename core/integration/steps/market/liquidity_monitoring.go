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
	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/liquidity-monitoring/*.json
	defaultLiquidityMonitoring          embed.FS
	defaultLiquidityMonitoringFileNames = []string{
		"defaults/liquidity-monitoring/default-parameters.json",
		"defaults/liquidity-monitoring/default-tenth.json",
	}
)

type liquidityMonitoring struct {
	config map[string]*vegapb.LiquidityMonitoringParameters
}

func newLiquidityMonitoring(unmarshaler *defaults.Unmarshaler) *liquidityMonitoring {
	liquidityMonitoring := &liquidityMonitoring{
		config: map[string]*vegapb.LiquidityMonitoringParameters{},
	}

	contentReaders := helpers.ReadAll(defaultLiquidityMonitoring, defaultLiquidityMonitoringFileNames)
	for name, contentReader := range contentReaders {
		liqMonParams, err := unmarshaler.UnmarshalLiquidityMonitoring(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default liquidity monitoring parameters %s: %v", name, err))
		}
		if err := liquidityMonitoring.Add(name, liqMonParams); err != nil {
			panic(fmt.Errorf("failed to add default liquidity monitoring %s: %v", name, err))
		}
	}

	return liquidityMonitoring
}

func (l *liquidityMonitoring) Add(name string, params *vegapb.LiquidityMonitoringParameters) error {
	l.config[name] = params
	return nil
}

func (l *liquidityMonitoring) GetType(name string) (*types.LiquidityMonitoringParameters, error) {
	config, ok := l.config[name]
	if !ok {
		return nil, fmt.Errorf("no liquidity monitoring \"%s\" registered", name)
	}
	return types.LiquidityMonitoringParametersFromProto(config)
}

func (l *liquidityMonitoring) Get(name string) (*vegapb.LiquidityMonitoringParameters, error) {
	config, ok := l.config[name]
	if !ok {
		return config, fmt.Errorf("no liquidity monitoring \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &vegapb.LiquidityMonitoringParameters{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy liquidity monitoring: %v", err))
	}
	return copyConfig, nil
}
