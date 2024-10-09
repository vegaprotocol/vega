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
)

func TheActivePAPIDShouldBeForMarket(
	engine Execution,
	market, expectedPAPID string,
) error {
	actualPAP, error := engine.GetMarketData(market)
	if error != nil {
		return error
	}
	if expectedPAPID == "none" && actualPAP.PAPState == nil {
		return nil
	}
	if expectedPAPID == "none" && actualPAP.PAPState != nil {
		return errPAPState(market, expectedPAPID, actualPAP.PAPState.Id)
	}
	if expectedPAPID != "none" && actualPAP.PAPState == nil {
		return errPAPState(market, expectedPAPID, "none")
	}
	if expectedPAPID != "none" && actualPAP.PAPState != nil && expectedPAPID != actualPAP.PAPState.Id {
		return errPAPState(market, expectedPAPID, actualPAP.PAPState.Id)
	}
	return nil
}

func errPAPState(market, expectedPAPID, actualPAPID string) error {
	return fmt.Errorf(fmt.Sprintf("unexpected pap id for market \"%s\"", market), expectedPAPID, actualPAPID)
}

func TheActivePAPOrderIDShouldBeForMarket(
	engine Execution,
	market, expectedPAPOrderID string,
) error {
	actualPAP, error := engine.GetMarketData(market)
	if error != nil {
		return error
	}
	if expectedPAPOrderID == "none" && actualPAP.PAPState == nil {
		return nil
	}
	if expectedPAPOrderID == "none" && actualPAP.PAPState != nil && actualPAP.PAPState.OrderId != nil {
		return errPAPOrderID(market, expectedPAPOrderID, *actualPAP.PAPState.OrderId)
	}
	if expectedPAPOrderID == "none" && actualPAP.PAPState != nil && actualPAP.PAPState.OrderId == nil {
		return nil
	}
	if expectedPAPOrderID != "none" && actualPAP.PAPState == nil {
		return errPAPOrderID(market, expectedPAPOrderID, "none")
	}
	if expectedPAPOrderID != "none" && actualPAP.PAPState != nil && actualPAP.PAPState.OrderId == nil {
		return errPAPOrderID(market, expectedPAPOrderID, "none")
	}
	if expectedPAPOrderID != "none" && actualPAP.PAPState != nil && actualPAP.PAPState.OrderId != nil && expectedPAPOrderID != *actualPAP.PAPState.OrderId {
		return errPAPOrderID(market, expectedPAPOrderID, *actualPAP.PAPState.OrderId)
	}
	return nil
}

func errPAPOrderID(market, expectedPAPID, actualPAPID string) error {
	return fmt.Errorf(fmt.Sprintf("unexpected pap id for market \"%s\"", market), expectedPAPID, actualPAPID)
}
