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
			VolumeDiscountFactors: types.Factors{
				Infra:     row.infraFactor(),
				Maker:     row.makerFactor(),
				Liquidity: row.liqFactor(),
			},
		}

		vbts = append(vbts, p)
	}
	tiers[volumeDiscountTierName] = vbts
	return nil
}

func parseVolumeDiscountTiersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"volume",
		"infra factor",
		"maker factor",
		"liquidity factor",
	}, []string{})
}

type volumeDiscountTiersRow struct {
	row RowWrapper
}

func (r volumeDiscountTiersRow) volume() *num.Uint {
	return r.row.MustUint("volume")
}

func (r volumeDiscountTiersRow) infraFactor() num.Decimal {
	return r.row.MustDecimal("infra factor")
}

func (r volumeDiscountTiersRow) makerFactor() num.Decimal {
	return r.row.MustDecimal("maker factor")
}

func (r volumeDiscountTiersRow) liqFactor() num.Decimal {
	return r.row.MustDecimal("liquidity factor")
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
			vdp.EndOfProgramTimestamp = time.Unix(row.closingTimestamp(), 0)
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

func parseFactorRow(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"maker factor",
		"liquidity factor",
		"infra factor",
	}, []string{})
}

type factorRow struct {
	r RowWrapper
}

func (f factorRow) party() types.PartyID {
	return types.PartyID(f.r.MustStr("party"))
}

func (f factorRow) maker() num.Decimal {
	return f.r.MustDecimal("maker factor")
}

func (f factorRow) liquidity() num.Decimal {
	return f.r.MustDecimal("liquidity factor")
}

func (f factorRow) infra() num.Decimal {
	return f.r.MustDecimal("infra factor")
}

func (f factorRow) String() string {
	return fmt.Sprintf("maker: %s, liquidity: %s, infra: %s", f.maker(), f.liquidity(), f.infra())
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

func (r volumeDiscountRow) closingTimestamp() int64 {
	return r.row.MustI64("closing timestamp")
}

func (r volumeDiscountRow) windowLength() uint64 {
	return r.row.MustU64("window length")
}

func PartiesHaveTheFollowingDiscountFactors(vde *volumediscount.Engine, table *godog.Table) error {
	for _, r := range parseFactorRow(table) {
		row := factorRow{
			r: r,
		}
		party := row.party()
		factors := vde.VolumeDiscountFactorForParty(types.PartyID(party))
		if !factors.Maker.Equal(row.maker()) || !factors.Liquidity.Equal(row.liquidity()) || !factors.Infra.Equal(row.infra()) {
			return fmt.Errorf(
				"factors for party %s don't match. Expected (%s), got (maker: %s, liquidity: %s, infra: %s)",
				party,
				row,
				factors.Maker,
				factors.Liquidity,
				factors.Infra,
			)
		}
	}
	return nil
}

func PartyHasTheFollowingDiscountInfraFactor(party, discountFactor string, vde *volumediscount.Engine) error {
	df := vde.VolumeDiscountFactorForParty(types.PartyID(party))
	df2, _ := num.DecimalFromString(discountFactor)
	if !df.Infra.Equal(df2) {
		return fmt.Errorf("%s has the discount factor of %s when we expected %s", party, df, df2)
	}
	return nil
}

func PartyHasTheFollowingDiscountMakerFactor(party, discountFactor string, vde *volumediscount.Engine) error {
	df := vde.VolumeDiscountFactorForParty(types.PartyID(party))
	df2, _ := num.DecimalFromString(discountFactor)
	if !df.Maker.Equal(df2) {
		return fmt.Errorf("%s has the discount factor of %s when we expected %s", party, df, df2)
	}
	return nil
}

func PartyHasTheFollowingDiscountLiquidityFactor(party, discountFactor string, vde *volumediscount.Engine) error {
	df := vde.VolumeDiscountFactorForParty(types.PartyID(party))
	df2, _ := num.DecimalFromString(discountFactor)
	if !df.Liquidity.Equal(df2) {
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

func AMMHasTheFollowingNotionalValue(exec Execution, vde *volumediscount.Engine, alias, value string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	// from this point, it's the same as for a normal party
	return PartyHasTheFollowingTakerNotional(id, value, vde)
}

func AMMHasTheFollowingDiscountInfraFactor(exec Execution, vde *volumediscount.Engine, alias, factor string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	return PartyHasTheFollowingDiscountInfraFactor(id, factor, vde)
}

func AMMHasTheFollowingDiscountMakerFactor(exec Execution, vde *volumediscount.Engine, alias, factor string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	return PartyHasTheFollowingDiscountMakerFactor(id, factor, vde)
}

func AMMHasTheFollowingDiscountLiquidityFactor(exec Execution, vde *volumediscount.Engine, alias, factor string) error {
	id, ok := exec.GetAMMSubAccountID(alias)
	if !ok {
		return fmt.Errorf("unknown vAMM alias %s", alias)
	}
	return PartyHasTheFollowingDiscountInfraFactor(id, factor, vde)
}
