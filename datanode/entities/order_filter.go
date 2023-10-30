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

package entities

import (
	"code.vegaprotocol.io/vega/protos/vega"
)

type OrderFilter struct {
	Statuses         []vega.Order_Status
	Types            []vega.Order_Type
	TimeInForces     []vega.Order_TimeInForce
	Reference        *string
	DateRange        *DateRange
	ExcludeLiquidity bool
	LiveOnly         bool
	PartyIDs         []string
	MarketIDs        []string
}

type StopOrderFilter struct {
	Statuses       []StopOrderStatus
	ExpiryStrategy []StopOrderExpiryStrategy
	DateRange      *DateRange
	PartyIDs       []string
	MarketIDs      []string
	LiveOnly       bool
}
