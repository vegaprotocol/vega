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
	//go:embed defaults/price-monitoring/*.json
	defaultPriceMonitoring          embed.FS
	defaultPriceMonitoringFileNames = []string{
		"defaults/price-monitoring/default-none.json",
		"defaults/price-monitoring/default-basic.json",
	}
)

type priceMonitoring struct {
	config map[string]*types.PriceMonitoringSettings
}

func newPriceMonitoring(unmarshaler *defaults.Unmarshaler) *priceMonitoring {
	priceMonitoring := &priceMonitoring{
		config: map[string]*types.PriceMonitoringSettings{},
	}

	contentReaders := helpers.ReadAll(defaultPriceMonitoring, defaultPriceMonitoringFileNames)
	for name, contentReader := range contentReaders {
		pm, err := unmarshaler.UnmarshalPriceMonitoring(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default price monitoring %s: %v", name, err))
		}
		if err := priceMonitoring.Add(name, pm); err != nil {
			panic(fmt.Errorf("failed to add default price monitoring %s: %v", name, err))
		}
	}

	return priceMonitoring
}

func (f *priceMonitoring) Add(
	name string,
	config *types.PriceMonitoringSettings,
) error {
	f.config[name] = config
	return nil
}

func (f *priceMonitoring) Get(name string) (*types.PriceMonitoringSettings, error) {
	config, ok := f.config[name]
	if !ok {
		return config, fmt.Errorf("no price monitoring \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &types.PriceMonitoringSettings{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy price monitoring: %v", err))
	}
	return copyConfig, nil
}
