# List of special types

Below is a list of types used in the docs and the possible values

## Position status

Possible values for `PositionStatus` are:

* POSITION_STATUS_UNSPECIFIED
* POSITION_STATUS_ORDERS_CLOSED
* POSITION_STATUS_CLOSED_OUT
* POSITION_STATUS_DISTRESSED

## Auction trigger

Possible values for `AuctionTrigger` are:

* AUCTION_TRIGGER_UNSPECIFIED
* AUCTION_TRIGGER_BATCH
* AUCTION_TRIGGER_OPENING
* AUCTION_TRIGGER_PRICE
* AUCTION_TRIGGER_LIQUIDITY
* AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET
* AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS
* AUCTION_TRIGGER_GOVERNANCE_SUSPENSION
* AUCTION_TRIGGER_LONG_BLOCK

## Trading mode

Possible values for `Market_TradingMode` are:

* TRADING_MODE_UNSPECIFIED
* TRADING_MODE_CONTINUOUS
* TRADING_MODE_BATCH_AUCTION
* TRADING_MODE_OPENING_AUCTION
* TRADING_MODE_MONITORING_AUCTION
* TRADING_MODE_NO_TRADING
* TRADING_MODE_SUSPENDED_VIA_GOVERNANCE
* TRADING_MODE_LONG_BLOCK_AUCTION

## Market state update

Possible values for `MarketStateUpdate` are:

* MARKET_STATE_UPDATE_TYPE_UNSPECIFIED
* MARKET_STATE_UPDATE_TYPE_TERMINATE
* MARKET_STATE_UPDATE_TYPE_SUSPEND
* MARKET_STATE_UPDATE_TYPE_RESUME

## Market state

Possible values for `Market_State` are:

* STATE_UNSPECIFIED
* STATE_PROPOSED
* STATE_REJECTED
* STATE_PENDING
* STATE_CANCELLED
* STATE_ACTIVE
* STATE_SUSPENDED
* STATE_CLOSED
* STATE_TRADING_TERMINATED
* STATE_SETTLED
* STATE_SUSPENDED_VIA_GOVERNANCE

## Order error

Possible values for `ORDER_ERROR` are:

* ORDER_ERROR_UNSPECIFIED
* ORDER_ERROR_INVALID_MARKET_ID
* ORDER_ERROR_INVALID_ORDER_ID
* ORDER_ERROR_OUT_OF_SEQUENCE
* ORDER_ERROR_INVALID_REMAINING_SIZE
* ORDER_ERROR_TIME_FAILURE
* ORDER_ERROR_REMOVAL_FAILURE
* ORDER_ERROR_INVALID_EXPIRATION_DATETIME
* ORDER_ERROR_INVALID_ORDER_REFERENCE
* ORDER_ERROR_EDIT_NOT_ALLOWED
* ORDER_ERROR_AMEND_FAILURE
* ORDER_ERROR_NOT_FOUND
* ORDER_ERROR_INVALID_PARTY_ID
* ORDER_ERROR_MARKET_CLOSED
* ORDER_ERROR_MARGIN_CHECK_FAILED
* ORDER_ERROR_MISSING_GENERAL_ACCOUNT
* ORDER_ERROR_INTERNAL_ERROR
* ORDER_ERROR_INVALID_SIZE
* ORDER_ERROR_INVALID_PERSISTENCE
* ORDER_ERROR_INVALID_TYPE
* ORDER_ERROR_SELF_TRADING
* ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES
* ORDER_ERROR_INCORRECT_MARKET_TYPE
* ORDER_ERROR_INVALID_TIME_IN_FORCE
* ORDER_ERROR_CANNOT_SEND_GFN_ORDER_DURING_AN_AUCTION
* ORDER_ERROR_CANNOT_SEND_GFA_ORDER_DURING_CONTINUOUS_TRADING
* ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
* ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
* ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
* ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
* ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
* ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
* ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION
* ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION
* ORDER_ERROR_MUST_BE_LIMIT_ORDER
* ORDER_ERROR_MUST_BE_GTT_OR_GTC
* ORDER_ERROR_WITHOUT_REFERENCE_PRICE
* ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE
* ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO
* ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE
* ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO
* ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
* ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER
* ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER
* ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER
* ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS
* ORDER_ERROR_TOO_MANY_PEGGED_ORDERS
* ORDER_ERROR_POST_ONLY_ORDER_WOULD_TRADE
* ORDER_ERROR_REDUCE_ONLY_ORDER_WOULD_NOT_REDUCE_POSITION
* ORDER_ERROR_ISOLATED_MARGIN_CHECK_FAILED
* ORDER_ERROR_PEGGED_ORDERS_NOT_ALLOWED_IN_ISOLATED_MARGIN_MODE
* ORDER_ERROR_PRICE_NOT_IN_TICK_SIZE
* ORDER_ERROR_PRICE_MUST_BE_LESS_THAN_OR_EQUAL_TO_MAX_PRICE


## Account type

Possible values for `ACCOUNT_TYPE` are:

* ACCOUNT_TYPE_UNSPECIFIED
* ACCOUNT_TYPE_INSURANCE
* ACCOUNT_TYPE_SETTLEMENT
* ACCOUNT_TYPE_MARGIN
* ACCOUNT_TYPE_GENERAL
* ACCOUNT_TYPE_FEES_INFRASTRUCTURE
* ACCOUNT_TYPE_FEES_LIQUIDITY
* ACCOUNT_TYPE_FEES_MAKER
* ACCOUNT_TYPE_BOND
* ACCOUNT_TYPE_EXTERNAL
* ACCOUNT_TYPE_GLOBAL_INSURANCE
* ACCOUNT_TYPE_GLOBAL_REWARD
* ACCOUNT_TYPE_PENDING_TRANSFERS
* ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES
* ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES
* ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES
* ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
* ACCOUNT_TYPE_HOLDING
* ACCOUNT_TYPE_LP_LIQUIDITY_FEES
* ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION
* ACCOUNT_TYPE_NETWORK_TREASURY
* ACCOUNT_TYPE_VESTING_REWARDS
* ACCOUNT_TYPE_VESTED_REWARDS
* ACCOUNT_TYPE_REWARD_RELATIVE_RETURN
* ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY
* ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING
* ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD
* ACCOUNT_TYPE_ORDER_MARGIN
* ACCOUNT_TYPE_REWARD_REALISED_RETURN
* ACCOUNT_TYPE_BUY_BACK_FEES
* ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL
* ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES

For in some debug output, particularly when the accounts types are used as part of an account ID, these constants will be shown as a numeric value, the mapping is as follows:

* 0: ACCOUNT_TYPE_UNSPECIFIED
* 1: ACCOUNT_TYPE_INSURANCE
* 2: ACCOUNT_TYPE_SETTLEMENT
* 3: ACCOUNT_TYPE_MARGIN
* 4: ACCOUNT_TYPE_GENERAL
* 5: ACCOUNT_TYPE_FEES_INFRASTRUCTURE
* 6: ACCOUNT_TYPE_FEES_LIQUIDITY
* 7: ACCOUNT_TYPE_FEES_MAKER
* 9: ACCOUNT_TYPE_BOND
* 10: ACCOUNT_TYPE_EXTERNAL
* 11: ACCOUNT_TYPE_GLOBAL_INSURANCE
* 12: ACCOUNT_TYPE_GLOBAL_REWARD
* 13: ACCOUNT_TYPE_PENDING_TRANSFERS
* 14: ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES
* 15: ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES
* 16: ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES
* 17: ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
* 18: ACCOUNT_TYPE_HOLDING
* 19: ACCOUNT_TYPE_LP_LIQUIDITY_FEES
* 20: ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION
* 21: ACCOUNT_TYPE_NETWORK_TREASURY
* 22: ACCOUNT_TYPE_VESTING_REWARDS
* 23: ACCOUNT_TYPE_VESTED_REWARDS
* 25: ACCOUNT_TYPE_REWARD_RELATIVE_RETURN
* 26: ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY
* 27: ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING
* 28: ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD
* 29: ACCOUNT_TYPE_ORDER_MARGIN
* 30: ACCOUNT_TYPE_REWARD_REALISED_RETURN
* 31: ACCOUNT_TYPE_BUY_BACK_FEES
* 32: ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL
* 33: ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES

## Transfer type

Possible values for `TRANSFER_TYPE` are:

* TRANSFER_TYPE_UNSPECIFIED
* TRANSFER_TYPE_LOSS
* TRANSFER_TYPE_WIN
* TRANSFER_TYPE_MTM_LOSS
* TRANSFER_TYPE_MTM_WIN
* TRANSFER_TYPE_MARGIN_LOW
* TRANSFER_TYPE_MARGIN_HIGH
* TRANSFER_TYPE_MARGIN_CONFISCATED
* TRANSFER_TYPE_MAKER_FEE_PAY
* TRANSFER_TYPE_MAKER_FEE_RECEIVE
* TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY
* TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE
* TRANSFER_TYPE_LIQUIDITY_FEE_PAY
* TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE
* TRANSFER_TYPE_BOND_LOW
* TRANSFER_TYPE_BOND_HIGH
* TRANSFER_TYPE_WITHDRAW
* TRANSFER_TYPE_DEPOSIT
* TRANSFER_TYPE_BOND_SLASHING
* TRANSFER_TYPE_REWARD_PAYOUT
* TRANSFER_TYPE_TRANSFER_FUNDS_SEND
* TRANSFER_TYPE_TRANSFER_FUNDS_DISTRIBUTE
* TRANSFER_TYPE_CLEAR_ACCOUNT
* TRANSFER_TYPE_CHECKPOINT_BALANCE_RESTORE
* TRANSFER_TYPE_SPOT
* TRANSFER_TYPE_HOLDING_LOCK
* TRANSFER_TYPE_HOLDING_RELEASE
* TRANSFER_TYPE_SUCCESSOR_INSURANCE_FRACTION
* TRANSFER_TYPE_LIQUIDITY_FEE_ALLOCATE
* TRANSFER_TYPE_LIQUIDITY_FEE_NET_DISTRIBUTE
* TRANSFER_TYPE_SLA_PENALTY_BOND_APPLY
* TRANSFER_TYPE_SLA_PENALTY_LP_FEE_APPLY
* TRANSFER_TYPE_LIQUIDITY_FEE_UNPAID_COLLECT
* TRANSFER_TYPE_SLA_PERFORMANCE_BONUS_DISTRIBUTE
* TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS
* TRANSFER_TYPE_PERPETUALS_FUNDING_WIN
* TRANSFER_TYPE_REWARDS_VESTED
* TRANSFER_TYPE_FEE_REFERRER_REWARD_PAY
* TRANSFER_TYPE_FEE_REFERRER_REWARD_DISTRIBUTE
* TRANSFER_TYPE_ORDER_MARGIN_LOW
* TRANSFER_TYPE_ORDER_MARGIN_HIGH
* TRANSFER_TYPE_ISOLATED_MARGIN_LOW
* TRANSFER_TYPE_ISOLATED_MARGIN_HIGH
* TRANSFER_TYPE_AMM_LOW
* TRANSFER_TYPE_AMM_HIGH
* TRANSFER_TYPE_AMM_RELEASE
* TRANSFER_TYPE_TREASURY_FEE_PAY
* TRANSFER_TYPE_BUY_BACK_FEE_PAY
* TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_PAY
* TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE


## Cancel AMM Method

Possible values for `CancelAMM_Method` are:

* METHOD_UNSPECIFIED
* METHOD_IMMEDIATE
* METHOD_REDUCE_ONLY

## AMM Status

Possible values for the `AMM_Status` are:

* STATUS_UNSPECIFIED
* STATUS_ACTIVE
* STATUS_REJECTED
* STATUS_CANCELLED
* STATUS_STOPPED
* STATUS_REDUCE_ONLY

## AMM Status Reason

Possible values for the `AMM_StatusReason` are:

* STATUS_REASON_UNSPECIFIED
* STATUS_REASON_CANCELLED_BY_PARTY
* STATUS_REASON_CANNOT_FILL_COMMITMENT
* STATUS_REASON_PARTY_ALREADY_OWNS_A_POOL
* STATUS_REASON_PARTY_CLOSED_OUT
* STATUS_REASON_MARKET_CLOSED
* STATUS_REASON_COMMITMENT_TOO_LOW
* STATUS_REASON_CANNOT_REBASE

## PropertyKey_Type

Possible values for `PropertyKey_Type` are:

* TYPE_UNSPECIFIED
* TYPE_INTEGER
* TYPE_STRING
* TYPE_EMPTY
* TYPE_BOOLEAN
* TYPE_DECIMAL
* TYPE_TIMESTAMP

## Condition_Operator

Possible values for `Condition_Operator` are:

* OPERATOR_UNSPECIFIED
* OPERATOR_EQUALS
* OPERATOR_LESS_THAN
* OPERATOR_LESS_THAN_OR_EQUAL
* OPERATOR_GREATER_THAN
* OPERATOR_GREATER_THAN_OR_EQUAL

## LiquidityFeeMethod

Possible values for `LiquidityFeeMethod` are:

* METHOD_UNSPECIFIED
* METHOD_CONSTANT
* METHOD_MARGINAL_COST
* METHOD_WEIGHTED_AVERAGE

## Price type

Possible values for price type are:

* last trade
* median
* weight

## Source weights

The source weight type takes the form:

`decimal,decimal,decimal,decimal`

And usually defaults to:

`0,0,0,0`

## Staleness tolerance

Staleness tolerance, analogous to [source weights](#Source-weights) takes the form:

`duration,duration,duration,duration`

Its defaults to either four times `1us`, or `1000s`
Valid inputs look like: `10s,1ms,0s,0s` or `1000s,0s,0s,0s`