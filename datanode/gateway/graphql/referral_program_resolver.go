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
	"errors"
	"math"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

type referralProgramResolver VegaResolverRoot

func (r *referralProgramResolver) Version(ctx context.Context, obj *vega.ReferralProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *referralProgramResolver) WindowLength(ctx context.Context, obj *vega.ReferralProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}

type currentReferralProgramResolver VegaResolverRoot

func (r *currentReferralProgramResolver) Version(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.Version > math.MaxInt {
		return 0, errors.New("version is too large")
	}

	return int(obj.Version), nil
}

func (r *currentReferralProgramResolver) WindowLength(ctx context.Context, obj *v2.ReferralProgram) (int, error) {
	if obj.WindowLength > math.MaxInt {
		return 0, errors.New("window length is too large")
	}

	return int(obj.WindowLength), nil
}
