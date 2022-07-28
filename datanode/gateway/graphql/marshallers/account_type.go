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

package marshallers

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/protos/vega"
	"github.com/99designs/gqlgen/graphql"
)

var (
	accountTypeToName = map[vega.AccountType]string{
		vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED:                "Unspecified",
		vega.AccountType_ACCOUNT_TYPE_INSURANCE:                  "Insurance",
		vega.AccountType_ACCOUNT_TYPE_SETTLEMENT:                 "Settlement",
		vega.AccountType_ACCOUNT_TYPE_MARGIN:                     "Margin",
		vega.AccountType_ACCOUNT_TYPE_GENERAL:                    "General",
		vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE:        "FeeInfrastructure",
		vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY:             "FeeLiquidity",
		vega.AccountType_ACCOUNT_TYPE_FEES_MAKER:                 "FeeMaker",
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

	nameToAccountType = map[string]vega.AccountType{
		"Unspecified":             vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED,
		"Insurance":               vega.AccountType_ACCOUNT_TYPE_INSURANCE,
		"Settlement":              vega.AccountType_ACCOUNT_TYPE_SETTLEMENT,
		"Margin":                  vega.AccountType_ACCOUNT_TYPE_MARGIN,
		"General":                 vega.AccountType_ACCOUNT_TYPE_GENERAL,
		"FeeInfrastructure":       vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		"FeeLiquidity":            vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		"FeeMaker":                vega.AccountType_ACCOUNT_TYPE_FEES_MAKER,
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
)

func MarshalAccountType(t vega.AccountType) graphql.ContextMarshaler {
	f := func(ctx context.Context, w io.Writer) error {
		s, ok := accountTypeToName[t]
		if !ok {
			return fmt.Errorf("unknown account type %v", t)
		}

		io.WriteString(w, strconv.Quote(s))
		return nil
	}
	return graphql.ContextWriterFunc(f)
}

func UnmarshalAccountType(ctx context.Context, v interface{}) (vega.AccountType, error) {
	s, ok := v.(string)
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("expected account type to be a string")
	}

	var ty vega.AccountType
	ty, ok = nameToAccountType[s]
	if !ok {
		return vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("failed to convert AccountType from GraphQL to Proto: %v", s)
	}

	return ty, nil
}
