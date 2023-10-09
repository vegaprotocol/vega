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
	//go:embed defaults/staking-tiers/*.json
	defaultStakingTiersConfigs embed.FS
	defaultStakingTiersNames   = []string{
		"defaults/staking-tiers/default.json",
	}
)

type stakingTiersConfig struct {
	config map[string][]*types.StakingTier
}

func newStakingTiersConfigs() *stakingTiersConfig {
	config := &stakingTiersConfig{
		config: map[string][]*types.StakingTier{},
	}

	contentReaders := helpers.ReadAll(defaultStakingTiersConfigs, defaultStakingTiersNames)
	for name, contentReader := range contentReaders {
		stakingTiersConfig, err := unmarshalStakingTiers(contentReader)
		if err != nil {
			panic(fmt.Errorf("couldn't unmarshal default staking tiers config %s: %v", name, err))
		}
		config.Add(name, stakingTiersConfig)
	}

	return config
}

func (f *stakingTiersConfig) Add(name string, stakingTiers []*types.StakingTier) {
	f.config[name] = stakingTiers
}

func (f *stakingTiersConfig) Get(name string) ([]*types.StakingTier, error) {
	stakingTiers, ok := f.config[name]
	if !ok {
		return nil, fmt.Errorf("no staking tiers configuration registered for name %q", name)
	}

	// Copy to avoid modification between tests.
	copyConfig := []*types.StakingTier{}
	if err := copier.Copy(&copyConfig, &stakingTiers); err != nil {
		return nil, fmt.Errorf("failed to deep copy staking tiers configuration: %v", err)
	}
	return copyConfig, nil
}

func unmarshalStakingTiers(r io.Reader) ([]*types.StakingTier, error) {
	proto := &vegapb.ReferralProgram{}
	unmarshaler := jsonpb.Unmarshaler{}
	err := unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	referralProgram := types.NewReferralProgramFromProto(proto)
	return referralProgram.StakingTiers, nil
}
