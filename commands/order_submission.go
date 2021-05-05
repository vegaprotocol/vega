package commands

import (
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

func CheckOrderSubmission(cmd *commandspb.OrderSubmission) Errors {
	errs := NewErrors()

	if len(cmd.MarketId) == 0 {
		errs.AddForProperty("order_submission.market_id", ErrIsRequired)
	}
	if cmd.Side == types.Side_SIDE_UNSPECIFIED {
		errs.AddForProperty("order_submission.side", ErrIsRequired)
	}

	if _, ok := types.Side_name[int32(cmd.Side)]; !ok {
		errs.AddForProperty("order_submission.side", ErrIsNotValid)
	}
	if cmd.Size <= 0 {
		errs.AddForProperty("order_submission.size", ErrMustBePositive)
	}

	return errs
}
//
//
// func (s *Svc) validateOrderSubmission(sub *commandspb.OrderSubmission) error {
// 	if this.MarketId == "" {
// 		return github_com_mwitkow_go_proto_validators.FieldError("MarketId", fmt.Errorf(`value '%v' must not be an empty string`, this.MarketId))
// 	}
// 	if this.PartyId == "" {
// 		return github_com_mwitkow_go_proto_validators.FieldError("PartyId", fmt.Errorf(`value '%v' must not be an empty string`, this.PartyId))
// 	}
// 	if !(this.Size > 0) {
// 		return github_com_mwitkow_go_proto_validators.FieldError("Size_", fmt.Errorf(`value '%v' must be greater than '0'`, this.Size))
// 	}
// 	if this.PeggedOrder != nil {
// 		if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(this.PeggedOrder); err != nil {
// 			return github_com_mwitkow_go_proto_validators.FieldError("PeggedOrder", err)
// 		}
// 	}
//
// 	if sub.Side == types.Side_SIDE_UNSPECIFIED {
// 		return ErrNoSide
// 	}
//
// 	if sub.Type == types.Order_TYPE_UNSPECIFIED {
// 		return ErrNoType
// 	}
//
// 	if sub.TimeInForce == types.Order_TIME_IN_FORCE_UNSPECIFIED {
// 		return ErrNoTimeInForce
// 	}
//
// 	if sub.TimeInForce == types.Order_TIME_IN_FORCE_GTT {
// 		if sub.ExpiresAt <= 0 {
// 			s.log.Error("invalid expiration time")
// 			return ErrInvalidExpirationDT
// 		}
// 	}
//
// 	if sub.TimeInForce != types.Order_TIME_IN_FORCE_GTT && sub.ExpiresAt != 0 {
// 		return ErrNonGTTOrderWithExpiry
// 	}
//
// 	if sub.Type == types.Order_TYPE_MARKET && sub.Price != 0 {
// 		return ErrInvalidPriceForMarketOrder
// 	}
// 	if sub.Type == types.Order_TYPE_MARKET &&
// 		(sub.TimeInForce != types.Order_TIME_IN_FORCE_FOK && sub.TimeInForce != types.Order_TIME_IN_FORCE_IOC) {
// 		return ErrInvalidTimeInForceForMarketOrder
// 	}
// 	if sub.Type == types.Order_TYPE_LIMIT && sub.Price == 0 &&
// 		sub.PeggedOrder == nil {
// 		return ErrInvalidPriceForLimitOrder
// 	}
// 	if sub.Type == types.Order_TYPE_NETWORK {
// 		return ErrUnAuthorizedOrderType
// 	}
//
// 	// Validation for pegged orders
// 	if sub.PeggedOrder != nil {
// 		if sub.Type != types.Order_TYPE_LIMIT {
// 			// All pegged orders must be LIMIT orders
// 			return ErrPeggedOrderMustBeLimitOrder
// 		}
//
// 		if sub.TimeInForce != types.Order_TIME_IN_FORCE_GTT && sub.TimeInForce != types.Order_TIME_IN_FORCE_GTC {
// 			// Pegged orders can only be GTC or GTT
// 			return ErrPeggedOrderMustBeGTTOrGTC
// 		}
//
// 		if sub.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
// 			// We must specify a valid reference
// 			return ErrPeggedOrderWithoutReferencePrice
// 		}
//
// 		if sub.Side == types.Side_SIDE_BUY {
// 			switch sub.PeggedOrder.Reference {
// 			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
// 				return ErrPeggedOrderBuyCannotReferenceBestAskPrice
// 			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
// 				if sub.PeggedOrder.Offset > 0 {
// 					return ErrPeggedOrderOffsetMustBeLessOrEqualToZero
// 				}
// 			case types.PeggedReference_PEGGED_REFERENCE_MID:
// 				if sub.PeggedOrder.Offset >= 0 {
// 					return ErrPeggedOrderOffsetMustBeLessThanZero
// 				}
// 			}
// 		} else {
// 			switch sub.PeggedOrder.Reference {
// 			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
// 				if sub.PeggedOrder.Offset < 0 {
// 					return ErrPeggedOrderOffsetMustBeGreaterOrEqualToZero
// 				}
// 			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
// 				return ErrPeggedOrderSellCannotReferenceBestBidPrice
// 			case types.PeggedReference_PEGGED_REFERENCE_MID:
// 				if sub.PeggedOrder.Offset <= 0 {
// 					return ErrPeggedOrderOffsetMustBeGreaterThanZero
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }
