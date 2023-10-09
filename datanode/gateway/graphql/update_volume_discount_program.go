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

	vega "code.vegaprotocol.io/vega/protos/vega"
)

type updateVolumeDiscountProgramResolver VegaResolverRoot

func (r *updateVolumeDiscountProgramResolver) Version(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int, error) {
	return int(obj.Changes.Version), nil
}

func (r *updateVolumeDiscountProgramResolver) ID(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (string, error) {
	return obj.Changes.Id, nil
}

func (r *updateVolumeDiscountProgramResolver) BenefitTiers(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) ([]*vega.VolumeBenefitTier, error) {
	return obj.Changes.BenefitTiers, nil
}

func (r *updateVolumeDiscountProgramResolver) EndOfProgramTimestamp(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int64, error) {
	return obj.Changes.EndOfProgramTimestamp, nil
}

func (r *updateVolumeDiscountProgramResolver) WindowLength(
	ctx context.Context, obj *vega.UpdateVolumeDiscountProgram,
) (int, error) {
	return int(obj.Changes.WindowLength), nil
}
