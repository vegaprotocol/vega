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
	//go:embed defaults/fees-config/*.json
	defaultFeesConfigs         embed.FS
	defaultFeesConfigFileNames = []string{
		"defaults/fees-config/default-none.json",
		"defaults/fees-config/ten-percent.json",
	}
)

type feesConfig struct {
	config map[string]*types.Fees
}

func newFeesConfig(unmarshaler *defaults.Unmarshaler) *feesConfig {
	config := &feesConfig{
		config: map[string]*types.Fees{},
	}

	contentReaders := helpers.ReadAll(defaultFeesConfigs, defaultFeesConfigFileNames)
	for name, contentReader := range contentReaders {
		feesConfig, err := unmarshaler.UnmarshalFeesConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default fees config %s: %v", name, err))
		}
		if err := config.Add(name, feesConfig); err != nil {
			panic(fmt.Errorf("failed to add default fees config %s: %v", name, err))
		}
	}

	return config
}

func (f *feesConfig) Add(name string, fees *types.Fees) error {
	f.config[name] = fees
	return nil
}

func (f *feesConfig) Get(name string) (*types.Fees, error) {
	fees, ok := f.config[name]
	if !ok {
		return fees, fmt.Errorf("no fees configuration \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &types.Fees{}
	if err := copier.Copy(copyConfig, fees); err != nil {
		panic(fmt.Errorf("failed to deep copy fees config: %v", err))
	}
	return copyConfig, nil
}
