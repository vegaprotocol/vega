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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func ValidatorsShouldHaveTheFollowingScores(
	broker *stubs.BrokerStub,
	table *godog.Table,
	epoch string,
) error {
	scores := broker.GetValidatorScores(epoch)

	for _, r := range parseValidatorScoreTable(table) {
		row := validatorScoreRow{row: r}
		validatorScore, ok := scores[row.NodeID()]

		score5DP, _ := num.DecimalFromString(validatorScore.ValidatorScore)
		normScore5DP, _ := num.DecimalFromString(validatorScore.NormalisedScore)

		if !ok {
			return errMismatchedScore(row.NodeID(), "validator score", row.ValidatorScore(), "0")
		}
		if score5DP.StringFixed(5) != row.ValidatorScore() {
			return errMismatchedScore(row.NodeID(), "validator score", row.ValidatorScore(), score5DP.StringFixed(5))
		}
		if normScore5DP.StringFixed(5) != row.NormalisedScore() {
			return errMismatchedScore(row.NodeID(), "validator normalised score", row.NormalisedScore(), normScore5DP.StringFixed(5))
		}
	}
	return nil
}

func errMismatchedScore(node, name, expectedScore, actualScore string) error {
	return formatDiff(
		fmt.Sprintf("(%s) did not match for node(%s)", name, node),
		map[string]string{
			name: expectedScore,
		},
		map[string]string{
			name: actualScore,
		},
	)
}

func parseValidatorScoreTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"node id",
		"validator score",
		"normalised score",
	}, nil)
}

type validatorScoreRow struct {
	row RowWrapper
}

func (r validatorScoreRow) NodeID() string {
	return r.row.MustStr("node id")
}

func (r validatorScoreRow) ValidatorScore() string {
	return r.row.MustStr("validator score")
}

func (r validatorScoreRow) NormalisedScore() string {
	return r.row.MustStr("normalised score")
}
