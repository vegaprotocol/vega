package market

import (
	"embed"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	//go:embed defaults/margin-calculator/*.json
	defaultMarginCalculators          embed.FS
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

	contentReaders := defaults.ReadAll(defaultMarginCalculators, defaultMarginCalculatorFileNames)
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
