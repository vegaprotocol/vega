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
	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/protos/vega"
)

var (
	//go:embed defaults/margin-calculator/*.json
	defaultMarginCalculators         embed.FS
	defaultMarginCalculatorFileNames = []string{
		"defaults/margin-calculator/default-margin-calculator.json",
		"defaults/margin-calculator/default-overkill-margin-calculator.json",
	}
)

type marginCalculators struct {
	config map[string]*types.MarginCalculator
}

func newMarginCalculators(unmarshaler *defaults.Unmarshaler) *marginCalculators {
	config := &marginCalculators{
		config: map[string]*types.MarginCalculator{},
	}

	contentReaders := helpers.ReadAll(defaultMarginCalculators, defaultMarginCalculatorFileNames)
	for name, contentReader := range contentReaders {
		marginCalculator, err := unmarshaler.UnmarshalMarginCalculator(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default margin calculator %s: %v", name, err))
		}
		if err := config.Add(name, marginCalculator); err != nil {
			panic(fmt.Errorf("failed to add default margin calculator %s: %v", name, err))
		}
	}

	return config
}

func (c *marginCalculators) Add(name string, calculator *types.MarginCalculator) error {
	c.config[name] = calculator
	return nil
}

func (c *marginCalculators) Get(name string) (*types.MarginCalculator, error) {
	calculator, ok := c.config[name]
	if !ok {
		return calculator, fmt.Errorf("no margin calculator \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &types.MarginCalculator{}
	if err := copier.Copy(copyConfig, calculator); err != nil {
		panic(fmt.Errorf("failed to deep copy margin calculator: %v", err))
	}
	return copyConfig, nil
}
