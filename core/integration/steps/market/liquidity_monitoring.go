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

	contentReaders := defaults.ReadAll(defaultLiquidityMonitoring, defaultLiquidityMonitoringFileNames)
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
