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

package common

import (
	"errors"
)

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market.
	ErrMarketClosed = errors.New("market closed")
	// ErrPartyDoNotExists signals that the party used does not exists.
	ErrPartyDoNotExists = errors.New("party does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed.
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds.
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrMissingGeneralAccountForParty ...
	ErrMissingGeneralAccountForParty = errors.New("missing general account for party")
	// ErrNotEnoughVolumeToZeroOutNetworkOrder ...
	ErrNotEnoughVolumeToZeroOutNetworkOrder = errors.New("not enough volume to zero out network order")
	// ErrInvalidAmendRemainQuantity signals incorrect remaining qty for a reduce by amend.
	ErrInvalidAmendRemainQuantity = errors.New("incorrect remaining qty for a reduce by amend")
	// ErrEmptyMarketID is returned if processed market has an empty id.
	ErrEmptyMarketID = errors.New("invalid market id (empty)")
	// ErrInvalidOrderType is returned if processed order has an invalid order type.
	ErrInvalidOrderType = errors.New("invalid order type")
	// ErrInvalidExpiresAtTime is returned if the expire time is before the createdAt time.
	ErrInvalidExpiresAtTime = errors.New("invalid expiresAt time")
	// ErrGFAOrderReceivedDuringContinuousTrading is returned is a gfa order hits the market when the market is in continuous trading state.
	ErrGFAOrderReceivedDuringContinuousTrading = errors.New("gfa order received during continuous trading")
	// ErrGFNOrderReceivedAuctionTrading is returned if a gfn order hits the market when in auction state.
	ErrGFNOrderReceivedAuctionTrading = errors.New("gfn order received during auction trading")
	// ErrIOCOrderReceivedAuctionTrading is returned if a ioc order hits the market when in auction state.
	ErrIOCOrderReceivedAuctionTrading = errors.New("ioc order received during auction trading")
	// ErrFOKOrderReceivedAuctionTrading is returned if a fok order hits the market when in auction state.
	ErrFOKOrderReceivedAuctionTrading = errors.New("fok order received during auction trading")
	// ErrUnableToReprice we are unable to get a price required to reprice.
	ErrUnableToReprice = errors.New("unable to reprice")
	// ErrOrderNotFound we cannot find the order in the market.
	ErrOrderNotFound = errors.New("unable to find the order in the market")
	// ErrTradingNotAllowed no trading related functionalities are allowed in the current state.
	ErrTradingNotAllowed = errors.New("trading not allowed")
	// ErrCommitmentSubmissionNotAllowed no commitment submission are permitted in the current state.
	ErrCommitmentSubmissionNotAllowed = errors.New("commitment submission not allowed")
	// ErrNotEnoughStake is returned when a LP update results in not enough commitment.
	ErrNotEnoughStake = errors.New("commitment submission rejected, not enough stake")
	// ErrPartyNotLiquidityProvider is returned when a LP update or cancel does not match an LP party.
	ErrPartyNotLiquidityProvider = errors.New("party is not a liquidity provider")
	// ErrPartyAlreadyLiquidityProvider is returned when a LP is submitted by a party which is already LP.
	ErrPartyAlreadyLiquidityProvider = errors.New("party is already a liquidity provider")
	// ErrCannotRejectMarketNotInProposedState.
	ErrCannotRejectMarketNotInProposedState = errors.New("cannot reject a market not in proposed state")
	// ErrCannotStateOpeningAuctionForMarketNotInProposedState.
	ErrCannotStartOpeningAuctionForMarketNotInProposedState = errors.New("cannot start the opening auction for a market not in proposed state")
	// ErrCannotRepriceDuringAuction.
	ErrCannotRepriceDuringAuction = errors.New("cannot reprice during auction")
	// ErrPartyInsufficientAssetBalance is returned when a party does not have sufficient balance of the required asset to perform an action.
	ErrPartyInsufficientAssetBalance                        = errors.New("party has insufficient balance in asset")
	ErrMaxStopOrdersPerPartyReached                         = errors.New("max stop orders per party reached")
	ErrStopOrderSubmissionNotAllowedWithoutExistingPosition = errors.New("stop order submission not allowed without existing position")
	ErrStopOrderSideNotClosingThePosition                   = errors.New("side used in stop order does not close the position")
	ErrStopOrderMustBeReduceOnly                            = errors.New("stop order must be reduce only")
	ErrStopOrderExpiryInThePast                             = errors.New("stop order expiry in the past")
	ErrStopOrderSizeOverridePercentageInvalid               = errors.New("stop order size override percentage value is invalid")
	ErrPartyHasNoExistingLiquidityProvision                 = errors.New("party has no existing liquidity provision")
	// ErrStopOrderNotAllowedDuringOpeningAuction is returned if a trader attempts to send a stop order to a market that is in opening auction.
	ErrStopOrderNotAllowedDuringOpeningAuction = errors.New("stop orders are not accepted during the opening auction")
	// ErrStopOrderNotAllowedSameExpiryTimeForOCO is returned if both sides of an OCO have the same expiry time.
	ErrStopOrderNotAllowedSameExpiry = errors.New("stop order OCOs must not have the same expiry time")
	// ErrStopOrderSizeOverrideNotSupportedForSpots is returned when a stop order is received with size override for a spot product.
	ErrStopOrderSizeOverrideNotSupportedForSpots = errors.New("stop order size override is not supported for spot product")
	ErrAMMCannotRebase                           = errors.New("not enough liquidity for AMM to rebase")
	// ErrInvalidOrderPrice is returned when an order is submitted to a capped future with a price > max price.
	ErrInvalidOrderPrice = errors.New("invalid order price")
	// ErrIsolatedMarginFullyCollateralised is returned when a party tries to switch margin modes on a fully collateralised market.
	ErrIsolatedMarginFullyCollateralised = errors.New("isolated margin not permitted on fully collateralised markets")
	// ErrSettlementDataOutOfRange is returned when a capped future receives settlement data that is outside of the acceptable range (either > max price, or neither 0 nor max for binary settlements).
	ErrSettlementDataOutOfRange = errors.New("settlement data is outside of the price cap")
	ErrAMMBoundsOutsidePriceCap = errors.New("an AMM bound is outside of the price cap")
)
