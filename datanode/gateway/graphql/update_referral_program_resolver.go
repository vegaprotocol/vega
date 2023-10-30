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

package gql

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/protos/vega"
)

type updateReferralProgramResolver VegaResolverRoot

func (u updateReferralProgramResolver) BenefitTiers(_ context.Context, obj *vega.UpdateReferralProgram) ([]*vega.BenefitTier, error) {
	return obj.Changes.BenefitTiers, nil
}

func (u updateReferralProgramResolver) EndOfProgramTimestamp(_ context.Context, obj *vega.UpdateReferralProgram) (int64, error) {
	endTime := time.Unix(obj.Changes.EndOfProgramTimestamp, 0)
	return endTime.UnixNano(), nil
}

func (u updateReferralProgramResolver) WindowLength(_ context.Context, obj *vega.UpdateReferralProgram) (int, error) {
	return int(obj.Changes.WindowLength), nil
}

func (u updateReferralProgramResolver) StakingTiers(_ context.Context, obj *vega.UpdateReferralProgram) ([]*vega.StakingTier, error) {
	return obj.Changes.StakingTiers, nil
}
