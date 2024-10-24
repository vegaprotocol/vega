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

package commands

import (
	"errors"
	"math"
	"math/big"

	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckOrderAmendment(cmd *commandspb.OrderAmendment) error {
	return checkOrderAmendment(cmd).ErrorOrNil()
}

func checkOrderAmendment(cmd *commandspb.OrderAmendment) Errors {
	var (
		errs       = NewErrors()
		isAmending bool
	)

	if cmd == nil {
		return errs.FinalAddForProperty("order_amendment", ErrIsRequired)
	}

	if len(cmd.OrderId) <= 0 {
		errs.AddForProperty("order_amendment.order_id", ErrIsRequired)
	} else if !IsVegaID(cmd.OrderId) {
		errs.AddForProperty("order_amendment.order_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.MarketId) <= 0 {
		errs.AddForProperty("order_amendment.market_id", ErrIsRequired)
	} else if !IsVegaID(cmd.MarketId) {
		errs.AddForProperty("order_amendment.market_id", ErrShouldBeAValidVegaID)
	}

	// Check we are not trying to amend to a GFA
	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_GFA {
		errs.AddForProperty("order_amendment.time_in_force", ErrCannotAmendToGFA)
	}

	// Check we are not trying to amend to a GFN
	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_GFN {
		errs.AddForProperty("order_amendment.time_in_force", ErrCannotAmendToGFN)
	}

	if cmd.Price != nil {
		isAmending = true
		if price, ok := big.NewInt(0).SetString(*cmd.Price, 10); !ok {
			errs.AddForProperty("order_amendment.price", ErrNotAValidInteger)
		} else if price.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("order_amendment.price", ErrIsRequired)
		}
	}

	if cmd.Size != nil {
		isAmending = true
		if *cmd.Size > math.MaxInt64/2 {
			errs.AddForProperty("order_amendment.size", ErrSizeIsTooLarge)
		}
	}

	if cmd.SizeDelta != 0 {
		if cmd.Size != nil {
			errs.AddForProperty("order_amendment.size_delta", ErrMustBeSetTo0IfSizeSet)
		}
		isAmending = true
	}

	if cmd.TimeInForce == types.Order_TIME_IN_FORCE_GTT {
		isAmending = true
		if cmd.ExpiresAt == nil {
			errs.AddForProperty(
				"order_amendment.time_in_force", ErrGTTOrderWithNoExpiry)
		}
	}

	if cmd.TimeInForce != types.Order_TIME_IN_FORCE_UNSPECIFIED {
		isAmending = true
		if _, ok := types.Order_TimeInForce_name[int32(cmd.TimeInForce)]; !ok {
			errs.AddForProperty("order_amendment.time_in_force", ErrIsNotValid)
		}
	}

	if cmd.PeggedReference != types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
		isAmending = true
		if _, ok := types.PeggedReference_name[int32(cmd.PeggedReference)]; !ok {
			errs.AddForProperty("order_amendment.pegged_reference", ErrIsNotValid)
		}
	}

	if cmd.ExpiresAt != nil && *cmd.ExpiresAt > 0 {
		isAmending = true
		if cmd.TimeInForce != types.Order_TIME_IN_FORCE_GTT &&
			cmd.TimeInForce != types.Order_TIME_IN_FORCE_UNSPECIFIED {
			errs.AddForProperty(
				"order_amendment.expires_at", ErrNonGTTOrderWithExpiry)
		}
	}

	if cmd.PeggedOffset != "" {
		isAmending = true
		if peggedOffset, ok := big.NewInt(0).SetString(cmd.PeggedOffset, 10); !ok {
			errs.AddForProperty("order_amendment.pegged_offset", ErrNotAValidInteger)
		} else if peggedOffset.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("order_amendment.pegged_offset", ErrMustBePositive)
		}
	}

	if cmd.VaultId != nil && !IsVegaID(*cmd.VaultId) {
		errs.AddForProperty("order_amendment.vault_id", ErrInvalidVaultID)
	}

	if !isAmending {
		errs.Add(errors.New("order_amendment does not amend anything"))
	}

	return errs
}
