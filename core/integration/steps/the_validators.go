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
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/delegation"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/libs/num"
)

func TheValidators(
	topology *stubs.TopologyStub,
	stakingAcountStub *stubs.StakingAccountStub,
	delegtionEngine *delegation.Engine,
	table *godog.Table,
) error {
	for _, r := range parseTable(table) {
		row := newValidatorRow(r)
		topology.AddValidator(row.id(), row.pubKey())

		amt, _ := num.UintFromString(row.stakingAccountBalance(), 10)
		stakingAcountStub.IncrementBalance(row.pubKey(), amt)
	}

	return nil
}

func newValidatorRow(r RowWrapper) validatorRow {
	row := validatorRow{
		row: r,
	}
	return row
}

func parseTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"staking account balance",
	}, []string{
		"pub_key",
	})
}

type validatorRow struct {
	row RowWrapper
}

func (r validatorRow) pubKey() string {
	pk, ok := r.row.StrB("pub_key")
	if !ok {
		return r.id()
	}
	return pk
}

func (r validatorRow) id() string {
	return r.row.MustStr("id")
}

func (r validatorRow) stakingAccountBalance() string {
	return r.row.MustStr("staking account balance")
}
