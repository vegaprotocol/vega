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

package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/volumediscount"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"
)

func VolumeDiscountProgramTiers(
	tiers map[string][]*types.VolumeBenefitTier,
	volumeDiscountTierName string,
	table *godog.Table,
) error {
	rows := parseVolumeDiscountTiersTable(table)
	vbts := make([]*types.VolumeBenefitTier, 0, len(rows))
	for _, r := range rows {
		row := volumeDiscountTiersRow{row: r}
		p := &types.VolumeBenefitTier{
			MinimumRunningNotionalTakerVolume: row.volume(),
			VolumeDiscountFactor:              row.factor(),
		}

		vbts = append(vbts, p)
	}
	tiers[volumeDiscountTierName] = vbts
	return nil
}

func parseVolumeDiscountTiersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"volume",
		"factor",
	}, []string{})
}

type volumeDiscountTiersRow struct {
	row RowWrapper
}

func (r volumeDiscountTiersRow) volume() *num.Uint {
	return r.row.MustUint("volume")
}

func (r volumeDiscountTiersRow) factor() num.Decimal {
	return r.row.MustDecimal("factor")
}

func VolumeDiscountProgram(
	vde *volumediscount.Engine,
	tiers map[string][]*types.VolumeBenefitTier,
	table *godog.Table,
) error {
	rows := parseVolumeDiscountTable(table)
	vdp := types.VolumeDiscountProgram{}

	for _, r := range rows {
		row := volumeDiscountRow{row: r}
		vdp.ID = row.id()
		vdp.WindowLength = row.windowLength()
		if row.closingTimestamp() == 0 {
			vdp.EndOfProgramTimestamp = time.Time{}
		} else {
			vdp.EndOfProgramTimestamp = time.Unix(int64(row.closingTimestamp()), 0)
		}
		tierName := row.tiers()
		if tier := tiers[tierName]; tier != nil {
			vdp.VolumeBenefitTiers = tier
		}
		vde.UpdateProgram(&vdp)
	}
	return nil
}

func parseVolumeDiscountTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"tiers",
		"closing timestamp",
		"window length",
	}, []string{})
}

type volumeDiscountRow struct {
	row RowWrapper
}

func (r volumeDiscountRow) id() string {
	return r.row.MustStr("id")
}

func (r volumeDiscountRow) tiers() string {
	return r.row.MustStr("tiers")
}

func (r volumeDiscountRow) closingTimestamp() uint64 {
	return r.row.MustU64("closing timestamp")
}

func (r volumeDiscountRow) windowLength() uint64 {
	return r.row.MustU64("window length")
}

func PartyHasTheFollowingDiscountFactor(party, discountFactor string, vde *volumediscount.Engine) error {
	df := vde.VolumeDiscountFactorForParty(types.PartyID(party))
	df2, _ := num.DecimalFromString(discountFactor)
	if !df.Equal(df2) {
		return fmt.Errorf("%s has the discount factor of %s when we expected %s", party, df, df2)
	}
	return nil
}

func PartyHasTheFollowingTakerNotional(party, notional string, vde *volumediscount.Engine) error {
	tn := vde.TakerNotionalForParty(types.PartyID(party))
	tn2, _ := num.DecimalFromString(notional)
	if !tn.Equal(tn2) {
		return fmt.Errorf("%s has the taker notional of %s when we expected %s", party, tn, tn2)
	}
	return nil
}
