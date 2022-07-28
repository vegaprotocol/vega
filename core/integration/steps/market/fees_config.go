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

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"

	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/fees-config/*.json
	defaultFeesConfigs         embed.FS
	defaultFeesConfigFileNames = []string{
		"defaults/fees-config/default-none.json",
	}
)

type feesConfig struct {
	config map[string]*types.Fees
}

func newFeesConfig(unmarshaler *defaults.Unmarshaler) *feesConfig {
	config := &feesConfig{
		config: map[string]*types.Fees{},
	}

	contentReaders := defaults.ReadAll(defaultFeesConfigs, defaultFeesConfigFileNames)
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
