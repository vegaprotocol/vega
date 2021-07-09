package market

import (
	"embed"
	"fmt"

	"github.com/jinzhu/copier"

	"code.vegaprotocol.io/vega/integration/steps/market/defaults"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
)

var (
	//go:embed defaults/oracle-config/*.json
	defaultOracleConfigs         embed.FS
	defaultOracleConfigFileNames = []string{
		"defaults/oracle-config/default-eth-for-future.json",
		"defaults/oracle-config/default-usd-for-future.json",
	}
)

type oracleConfigs struct {
	config map[string]*OracleConfig
}

type OracleConfig struct {
	Spec    *oraclesv1.OracleSpec
	Binding *types.OracleSpecToFutureBinding
}

func newOracleSpecs(unmarshaler *defaults.Unmarshaler) *oracleConfigs {
	specs := &oracleConfigs{
		config: map[string]*OracleConfig{},
	}

	contentReaders := defaults.ReadAll(defaultOracleConfigs, defaultOracleConfigFileNames)
	for name, contentReader := range contentReaders {
		future, err := unmarshaler.UnmarshalOracleConfig(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default oracle config %s: %v", name, err))
		}
		if err := specs.Add(name, future.OracleSpec, future.OracleSpecBinding); err != nil {
			panic(fmt.Errorf("failed to add default oracle config %s: %v", name, err))
		}
	}

	return specs
}

func (f *oracleConfigs) Add(
	name string,
	spec *oraclesv1.OracleSpec,
	binding *types.OracleSpecToFutureBinding,
) error {
	f.config[name] = &OracleConfig{
		Spec:    spec,
		Binding: binding,
	}
	return nil
}

func (f *oracleConfigs) Get(name string) (*OracleConfig, error) {
	config, ok := f.config[name]
	if !ok {
		return config, fmt.Errorf("no oracle spec \"%s\" registered", name)
	}
	// Copy to avoid modification between tests.
	copyConfig := &OracleConfig{}
	if err := copier.Copy(copyConfig, config); err != nil {
		panic(fmt.Errorf("failed to deep copy oracle config: %v", err))
	}
	return copyConfig, nil
}
