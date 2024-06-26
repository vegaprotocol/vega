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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckOracleDataSubmission(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilOracleDataSubmissionFails)
	t.Run("Submitting an oracle data without payload fails", testOracleDataSubmissionWithoutPayloadFails)
	t.Run("Submitting an oracle data with payload succeeds", testOracleDataSubmissionWithPayloadSucceeds)
	t.Run("Submitting an oracle data without source fails", testOracleDataSubmissionWithoutSourceFails)
	t.Run("Submitting an oracle data with invalid source fails", testOracleDataSubmissionWithInvalidSourceFails)
	t.Run("Submitting an oracle data with source succeeds", testOracleDataSubmissionWithSourceSucceeds)
}

func testNilOracleDataSubmissionFails(t *testing.T) {
	err := checkOracleDataSubmission(nil)

	assert.Contains(t, err.Get("oracle_data_submission"), commands.ErrIsRequired)
}

func testOracleDataSubmissionWithoutPayloadFails(t *testing.T) {
	err := checkOracleDataSubmission(&commandspb.OracleDataSubmission{})
	assert.Contains(t, err.Get("oracle_data_submission.payload"), commands.ErrIsRequired)
}

func testOracleDataSubmissionWithPayloadSucceeds(t *testing.T) {
	err := checkOracleDataSubmission(&commandspb.OracleDataSubmission{
		Payload: []byte("0xDEADBEEF"),
	})
	assert.NotContains(t, err.Get("oracle_data_submission.payload"), commands.ErrIsRequired)
}

func testOracleDataSubmissionWithoutSourceFails(t *testing.T) {
	err := checkOracleDataSubmission(&commandspb.OracleDataSubmission{})
	assert.Contains(t, err.Get("oracle_data_submission.source"), commands.ErrIsRequired)
}

func testOracleDataSubmissionWithInvalidSourceFails(t *testing.T) {
	err := checkOracleDataSubmission(&commandspb.OracleDataSubmission{
		Source: commandspb.OracleDataSubmission_OracleSource(-42),
	})
	assert.Contains(t, err.Get("oracle_data_submission.source"), commands.ErrIsNotValid)
}

func testOracleDataSubmissionWithSourceSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value commandspb.OracleDataSubmission_OracleSource
	}{
		{
			msg:   "with Open Oracle source",
			value: commandspb.OracleDataSubmission_ORACLE_SOURCE_OPEN_ORACLE,
		}, {
			msg:   "with JSON source",
			value: commandspb.OracleDataSubmission_ORACLE_SOURCE_JSON,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkOracleDataSubmission(&commandspb.OracleDataSubmission{
				Source: tc.value,
			})
			assert.NotContains(t, err.Get("oracle_data_submission.source"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("oracle_data_submission.source"), commands.ErrIsNotValid)
		})
	}
}

func checkOracleDataSubmission(cmd *commandspb.OracleDataSubmission) commands.Errors {
	err := commands.CheckOracleDataSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
