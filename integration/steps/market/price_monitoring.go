package market

import (
	"embed"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/proto"
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

	contentReaders := defaults.ReadAll(defaultPriceMonitoring, defaultPriceMonitoringFileNames)
	for name, contentReader := range contentReaders {
		riskModel, err := unmarshaler.UnmarshalPriceMonitoring(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default risk model %s: %v", name, err))
		}
		if err := priceMonitoring.Add(name, riskModel); err != nil {
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
