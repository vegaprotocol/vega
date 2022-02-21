package marshallers

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/protos/vega"
	"github.com/99designs/gqlgen/graphql"
)

var accountTypeToName = map[vega.AccountType]string{
	vega.AccountType_ACCOUNT_TYPE_INSURANCE:                  "Insurance",
	vega.AccountType_ACCOUNT_TYPE_SETTLEMENT:                 "Settlement",
	vega.AccountType_ACCOUNT_TYPE_MARGIN:                     "Margin",
	vega.AccountType_ACCOUNT_TYPE_GENERAL:                    "General",
	vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE:        "FeeInfrastructure",
	vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY:             "FeeLiquidity",
	vega.AccountType_ACCOUNT_TYPE_FEES_MAKER:                 "FeeMaker",
	vega.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW:              "LockWithdraw",
	vega.AccountType_ACCOUNT_TYPE_BOND:                       "Bond",
	vega.AccountType_ACCOUNT_TYPE_EXTERNAL:                   "External",
	vega.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE:           "GlobalInsurance",
	vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD:              "GlobalReward",
	vega.AccountType_ACCOUNT_TYPE_PENDING_TRANSFERS:          "PendingTransfers",
	vega.AccountType_ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES:     "RewardTakerPaidFees",
	vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES: "RewardMakerReceivedFees",
	vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES:    "RewardLpReceivedFees",
	vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS:    "RewardMarketProposers",
}

var nameToAccountType = map[string]vega.AccountType{
	"Insurance":               vega.AccountType_ACCOUNT_TYPE_INSURANCE,
	"Settlement":              vega.AccountType_ACCOUNT_TYPE_SETTLEMENT,
	"Margin":                  vega.AccountType_ACCOUNT_TYPE_MARGIN,
	"General":                 vega.AccountType_ACCOUNT_TYPE_GENERAL,
	"FeeInfrastructure":       vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
	"FeeLiquidity":            vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
	"FeeMaker":                vega.AccountType_ACCOUNT_TYPE_FEES_MAKER,
	"LockWithdraw":            vega.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW,
	"Bond":                    vega.AccountType_ACCOUNT_TYPE_BOND,
	"External":                vega.AccountType_ACCOUNT_TYPE_EXTERNAL,
	"GlobalInsurance":         vega.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE,
	"GlobalReward":            vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
	"PendingTransfers":        vega.AccountType_ACCOUNT_TYPE_PENDING_TRANSFERS,
	"RewardTakerPaidFees":     vega.AccountType_ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES,
	"RewardMakerReceivedFees": vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES,
	"RewardLpReceivedFees":    vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES,
	"RewardMarketProposers":   vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS,
}

func MarshalAccountType(t vega.AccountType) graphql.ContextMarshaler {
	if t == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return graphql.Null
	}

	f := func(ctx context.Context, w io.Writer) error {
		s, ok := accountTypeToName[t]
		if !ok {
			return fmt.Errorf("Unknown account type %v", t)
		}

		io.WriteString(w, strconv.Quote(s))
		return nil
	}
	return graphql.ContextWriterFunc(f)
}

func UnmarshalAccountType(ctx context.Context, v interface{}) (vega.AccountType, error) {
	s, ok := v.(string)
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("Expected account type to be a string")
	}

	var ty vega.AccountType
	ty, ok = nameToAccountType[s]
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert AccountType from GraphQL to Proto: %v", s)
	}

	return ty, nil
}
