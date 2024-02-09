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
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"golang.org/x/exp/maps"
)

type ErroneousRow interface {
	ExpectError() bool
	Error() string
	Reference() string
}

func DebugTxErrors(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DEBUGGING ALL TRANSACTION ERRORS")
	data := broker.GetTxErrors()
	for _, e := range data {
		p := e.Proto()
		log.Infof("TxError: %s\n", p.String())
	}
}

func DebugLPSTxErrors(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DEBUGGING LP SUBMISSION ERRORS")
	data := broker.GetLPSErrors()
	for _, e := range data {
		p := e.Proto()
		log.Infof("LP Submission error: %s - LP: %#v\n", p.String(), p.GetLiquidityProvisionSubmission)
	}
}

// checkExpectedError checks if expected error has been returned,
// if no expectation has been set a regular error check is carried out,
// unexpectedErrDetail is an optional parameter that can be used to return a more detailed error when an unexpected error is encountered.
func checkExpectedError(row ErroneousRow, returnedErr, unexpectedErrDetail error) error {
	if row.ExpectError() && returnedErr == nil {
		return fmt.Errorf("action on %q should have fail", row.Reference())
	}

	if returnedErr != nil {
		if !row.ExpectError() {
			if unexpectedErrDetail != nil {
				return unexpectedErrDetail
			}
			return fmt.Errorf("action on %q has failed: %s", row.Reference(), returnedErr.Error())
		}

		if row.Error() != returnedErr.Error() {
			return formatDiff(fmt.Sprintf("action on %q is failing as expected but not with the expected error message", row.Reference()),
				map[string]string{
					"error": row.Error(),
				},
				map[string]string{
					"error": returnedErr.Error(),
				},
			)
		}
	}
	return nil
}

func formatDiff(msg string, expected, got map[string]string) error {
	var expectedStr strings.Builder
	var gotStr strings.Builder
	padding := findLongestKeyLen(expected) + 1
	formatStr := "\n\t\t%-*s(%s)"
	for name, value := range expected {
		_, _ = fmt.Fprintf(&expectedStr, formatStr, padding, name, value)
		_, _ = fmt.Fprintf(&gotStr, formatStr, padding, name, got[name])
	}

	return fmt.Errorf("%s\n\texpected:%s\n\tgot:%s",
		msg,
		expectedStr.String(),
		gotStr.String(),
	)
}

func findLongestKeyLen(expected map[string]string) int {
	keys := maps.Keys(expected)
	maxLen := 0
	for i := range keys {
		iLen := len(keys[i])
		if iLen > maxLen {
			maxLen = iLen
		}
	}
	return maxLen
}

func u64ToS(n uint64) string {
	return strconv.FormatUint(n, 10)
}

func u64SToS(ns []uint64) string {
	ss := []string{}
	for _, n := range ns {
		ss = append(ss, u64ToS(n))
	}
	return strings.Join(ss, " ")
}

func i64ToS(n int64) string {
	return strconv.FormatInt(n, 10)
}

func errOrderNotFound(reference string, party string, err error) error {
	return fmt.Errorf("order not found for party(%s) with reference(%s): %v", party, reference, err)
}

func errMarketDataNotFound(marketID string, err error) error {
	return fmt.Errorf("market data not found for market(%v): %s", marketID, err.Error())
}

type CancelOrderError struct {
	reference string
	request   commandspb.OrderCancellation
	Err       error
}

func (c CancelOrderError) Error() string {
	return fmt.Sprintf("failed to cancel order [%v] with reference %s: %v", c.request, c.reference, c.Err)
}

func (c *CancelOrderError) Unwrap() error { return c.Err }

type SubmitOrderError struct {
	reference string
	request   commandspb.OrderSubmission
	Err       error
}

func (s SubmitOrderError) Error() string {
	return fmt.Sprintf("failed to submit order [%v] with reference %s: %v", s.request, s.reference, s.Err)
}

func (s *SubmitOrderError) Unwrap() error { return s.Err }

func errOrderEventsNotFound(party, marketID string, side types.Side, size, price uint64) error {
	return fmt.Errorf("no matching order event found %v, %v, %v, %v, %v", party, marketID, side.String(), size, price)
}

func errNoWatchersSpecified(netparam string) error {
	return fmt.Errorf("no watchers specified for network parameter `%v`", netparam)
}

func errStopOrderEventsNotFound(party, marketID string, status types.StopOrder_Status) error {
	return fmt.Errorf("no matching stop order event found %v, %v, %v", party, marketID, status.String())
}
