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
	"fmt"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/integration/stubs"
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
