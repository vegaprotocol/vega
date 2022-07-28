// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"

	"github.com/cucumber/godog"
)

func RegisterAsset(tbl *godog.Table, asset *stubs.AssetStub) error {
	rows := StrictParseTable(tbl, []string{
		"id",
		"decimal places",
	}, nil)
	for _, row := range rows {
		asset.Register(
			row.MustStr("id"),
			row.MustU64("decimal places"),
		)
	}
	return nil
}
