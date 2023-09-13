package referral

import (
	"embed"
	"fmt"
	"io"

	"code.vegaprotocol.io/vega/core/integration/steps/helpers"
	"code.vegaprotocol.io/vega/core/types"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jinzhu/copier"
)

var (
	//go:embed defaults/benefit-tiers/*.json
	defaultBenefitTiersConfigs embed.FS
	defaultBenefitTiersNames   = []string{
		"defaults/benefit-tiers/default.json",
	}
)

type benefitTiersConfig struct {
	config map[string][]*types.BenefitTier
}

func newBenefitTiersConfigs() *benefitTiersConfig {
	config := &benefitTiersConfig{
		config: map[string][]*types.BenefitTier{},
	}

	contentReaders := helpers.ReadAll(defaultBenefitTiersConfigs, defaultBenefitTiersNames)
	for name, contentReader := range contentReaders {
		benefitTiersConfig, err := unmarshalBenefitTiers(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default benefit tiers config %s: %v", name, err))
		}
		config.Add(name, benefitTiersConfig)
	}

	return config
}

func (f *benefitTiersConfig) Add(name string, benefitTiers []*types.BenefitTier) {
	f.config[name] = benefitTiers
	return
}

func (f *benefitTiersConfig) Get(name string) ([]*types.BenefitTier, error) {
	benefitTiers, ok := f.config[name]
	if !ok {
		return nil, fmt.Errorf("no benefit tiers configuration registered for name %q", name)
	}

	// Copy to avoid modification between tests.
	copyConfig := []*types.BenefitTier{}
	if err := copier.Copy(&copyConfig, &benefitTiers); err != nil {
		return nil, fmt.Errorf("failed to deep copy benefit tiers configuration: %v", err)
	}
	return copyConfig, nil
}

func unmarshalBenefitTiers(r io.Reader) ([]*types.BenefitTier, error) {
	proto := &vegapb.ReferralProgram{}
	unmarshaler := jsonpb.Unmarshaler{}
	err := unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	referralProgram := types.NewReferralProgramFromProto(proto)
	return referralProgram.BenefitTiers, nil
}
