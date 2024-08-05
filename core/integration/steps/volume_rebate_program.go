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
	"code.vegaprotocol.io/vega/core/volumerebate"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func VolumeRebateProgramTiers(
	tiers map[string][]*types.VolumeRebateBenefitTier,
	volumeRebateTierName string,
	table *godog.Table,
) error {
	rows := parseVolumeRebateTiersTable(table)
	vbts := make([]*types.VolumeRebateBenefitTier, 0, len(rows))
	for _, r := range rows {
		row := volumeRebateTiersRow{row: r}
		p := &types.VolumeRebateBenefitTier{
			MinimumPartyMakerVolumeFraction: row.fraction(),
			AdditionalMakerRebate:           row.rebate(),
		}

		vbts = append(vbts, p)
	}
	tiers[volumeRebateTierName] = vbts
	return nil
}

func parseVolumeRebateTiersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"fraction",
		"rebate",
	}, []string{})
}

type volumeRebateTiersRow struct {
	row RowWrapper
}

func (r volumeRebateTiersRow) fraction() num.Decimal {
	return r.row.MustDecimal("fraction")
}

func (r volumeRebateTiersRow) rebate() num.Decimal {
	return r.row.MustDecimal("rebate")
}

func VolumeRebateProgram(
	vde *volumerebate.Engine,
	tiers map[string][]*types.VolumeRebateBenefitTier,
	table *godog.Table,
) error {
	rows := parseVolumeRebateTable(table)
	vdp := types.VolumeRebateProgram{}

	for _, r := range rows {
		row := volumeRebateRow{row: r}
		vdp.ID = row.id()
		vdp.WindowLength = row.windowLength()
		if row.closingTimestamp() == 0 {
			vdp.EndOfProgramTimestamp = time.Time{}
		} else {
			vdp.EndOfProgramTimestamp = time.Unix(row.closingTimestamp(), 0)
		}
		tierName := row.tiers()
		if tier := tiers[tierName]; tier != nil {
			vdp.VolumeRebateBenefitTiers = tier
		}
		vde.UpdateProgram(&vdp)
	}
	return nil
}

func parseVolumeRebateTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"tiers",
		"closing timestamp",
		"window length",
	}, []string{})
}

type volumeRebateRow struct {
	row RowWrapper
}

func (r volumeRebateRow) id() string {
	return r.row.MustStr("id")
}

func (r volumeRebateRow) tiers() string {
	return r.row.MustStr("tiers")
}

func (r volumeRebateRow) closingTimestamp() int64 {
	return r.row.MustI64("closing timestamp")
}

func (r volumeRebateRow) windowLength() uint64 {
	return r.row.MustU64("window length")
}

func PartyHasTheFollowingRebate(party, RebateFactor string, vde *volumerebate.Engine) error {
	df := vde.VolumeRebateFactorForParty(types.PartyID(party))
	df2, _ := num.DecimalFromString(RebateFactor)
	if !df.Equal(df2) {
		return fmt.Errorf("%s has the Rebate of %s when we expected %s", party, df, df2)
	}
	return nil
}

func PartyHasTheFollowingMakerVolumeFraction(party, fraction string, vde *volumerebate.Engine) error {
	tn := vde.MakerVolumeFractionForParty(types.PartyID(party))
	tn2, _ := num.DecimalFromString(fraction)
	if !tn.Equal(tn2) {
		return fmt.Errorf("%s has the maker volume fraction of %s when we expected %s", party, tn, tn2)
	}
	return nil
}

func AMMHasTheFollowingMakerVolumeFraction(exec Execution, vde *volumerebate.Engine, alias, value string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	// from this point, it's the same as for a normal party
	return PartyHasTheFollowingMakerVolumeFraction(id, value, vde)
}

func AMMHasTheFollowingRebate(exec Execution, vde *volumerebate.Engine, alias, factor string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	return PartyHasTheFollowingRebate(id, factor, vde)
}
