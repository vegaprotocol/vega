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
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func PartiesAvailableFeeDiscounts(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseTransferFeeDiscountTable(table) {

		asset := r.MustStr("asset")
		party := r.MustStr("party")
		actual := engine.AvailableFeeDiscount(asset, party)
		expected := r.MustStr("available discount")
		if expected != actual.String() {
			errs = append(errs, errors.New(r.MustStr("party")+" expected "+expected+" but got "+actual.String()))
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseTransferFeeDiscountTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"asset",
		"available discount",
	}, []string{})
}

func PartiesSubmitTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseOneOffTransferTable(table) {
		transfer, _ := rowToOneOffTransfer(r)
		err := engine.TransferFunds(context.Background(), &types.TransferFunds{
			Kind:   types.TransferCommandKindOneOff,
			OneOff: transfer,
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseOneOffTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id", "from", "from_account_type", "to", "to_account_type", "asset", "amount", "delivery_time",
	}, []string{"market", "error"})
}

func rowToOneOffTransfer(r RowWrapper) (*types.OneOffTransfer, error) {
	id := r.MustStr("id")
	from := r.MustStr("from")
	fromAccountType := r.MustStr("from_account_type")
	fromAT := proto.AccountType_value[fromAccountType]
	to := r.MustStr("to")
	toAccuontType := r.MustStr("to_account_type")
	toAT := proto.AccountType_value[toAccuontType]
	asset := r.MustStr("asset")
	amount := r.MustStr("amount")
	amountUint, _ := num.UintFromString(amount, 10)
	deliveryTime, err := time.Parse("2006-01-02T15:04:05Z", r.MustStr("delivery_time"))
	if err != nil {
		return nil, err
	}

	oneOff := &types.OneOffTransfer{
		TransferBase: &types.TransferBase{
			ID:              id,
			From:            from,
			FromAccountType: types.AccountType(fromAT),
			To:              to,
			ToAccountType:   types.AccountType(toAT),
			Asset:           asset,
			Amount:          amountUint,
		},
		DeliverOn: &deliveryTime,
	}
	return oneOff, nil
}

func PartiesSubmitRecurringTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseRecurringTransferTable(table) {
		transfer := rowToRecurringTransfer(r)
		err := engine.TransferFunds(context.Background(), &types.TransferFunds{
			Kind:      types.TransferCommandKindRecurring,
			Recurring: transfer,
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseRecurringTransferTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id", "from", "from_account_type", "to", "to_account_type", "asset", "amount", "start_epoch", "end_epoch", "factor",
	}, []string{"metric", "metric_asset", "markets", "lock_period", "window_length", "entity_scope", "individual_scope", "teams", "ntop", "staking_requirement", "notional_requirement", "distribution_strategy", "ranks", "error"})
}

func rowToRecurringTransfer(r RowWrapper) *types.RecurringTransfer {
	id := r.MustStr("id")
	from := r.MustStr("from")
	fromAccountType := r.MustStr("from_account_type")
	fromAT := proto.AccountType_value[fromAccountType]
	to := r.MustStr("to")
	toAccuontType := r.MustStr("to_account_type")
	toAT := proto.AccountType_value[toAccuontType]
	asset := r.MustStr("asset")
	amount := r.MustStr("amount")
	amountUint, _ := num.UintFromString(amount, 10)
	startEpoch, _ := num.UintFromString(r.MustStr("start_epoch"), 10)
	endEpoch := r.MustStr("end_epoch")
	var endEpochPtr *uint64
	if len(endEpoch) > 0 {
		endEpochUint, _ := num.UintFromString(r.MustStr("end_epoch"), 10)
		endEpochUint64 := endEpochUint.Uint64()
		endEpochPtr = &endEpochUint64
	}

	var dispatchStrategy *proto.DispatchStrategy
	if len(r.Str("metric")) > 0 {
		mkts := strings.Split(r.MustStr("markets"), ",")
		if len(mkts) == 1 && mkts[0] == "" {
			mkts = []string{}
		}
		lockPeriod := uint64(1)
		if r.HasColumn("lock_period") {
			lockPeriod = r.U64("lock_period")
		}
		windowLength := uint64(1)
		if r.HasColumn("window_length") {
			windowLength = r.U64("window_length")
		}

		distributionStrategy := proto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA
		var ranks []*proto.Rank
		if r.HasColumn("distribution_strategy") {
			distStrat := r.Str("distribution_strategy")
			if distStrat == "PRO_RATA" {
				distributionStrategy = proto.DistributionStrategy_DISTRIBUTION_STRATEGY_PRO_RATA
			} else if distStrat == "RANK" {
				distributionStrategy = proto.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK
				rankList := strings.Split(r.MustStr("ranks"), ",")
				ranks = make([]*proto.Rank, 0, len(rankList))
				for _, r := range rankList {
					rr := strings.Split(r, ":")
					startRank, _ := strconv.ParseUint(rr[0], 10, 32)
					shareRatio, _ := strconv.ParseUint(rr[1], 10, 32)
					ranks = append(ranks, &proto.Rank{StartRank: uint32(startRank), ShareRatio: uint32(shareRatio)})
				}
			}
		}

		entityScope := proto.EntityScope_ENTITY_SCOPE_INDIVIDUALS
		if r.HasColumn("entity_scope") {
			scope := r.Str("entity_scope")
			if scope == "INDIVIDUALS" {
				entityScope = proto.EntityScope_ENTITY_SCOPE_INDIVIDUALS
			} else if scope == "TEAMS" {
				entityScope = proto.EntityScope_ENTITY_SCOPE_TEAMS
			}
		}

		indiScope := proto.IndividualScope_INDIVIDUAL_SCOPE_UNSPECIFIED
		if entityScope == proto.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
			indiScope = proto.IndividualScope_INDIVIDUAL_SCOPE_ALL
			if r.HasColumn("individual_scope") {
				indiScopeStr := r.Str("individual_scope")
				if indiScopeStr == "ALL" {
					indiScope = proto.IndividualScope_INDIVIDUAL_SCOPE_ALL
				} else if indiScopeStr == "IN_TEAM" {
					indiScope = proto.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM
				} else if indiScopeStr == "NOT_IN_TEAM" {
					indiScope = proto.IndividualScope_INDIVIDUAL_SCOPE_NOT_IN_TEAM
				}
			}
		}

		teams := []string{}
		ntop := ""
		if entityScope == proto.EntityScope_ENTITY_SCOPE_TEAMS {
			if r.HasColumn("teams") {
				teams = strings.Split(r.MustStr("teams"), ",")
				if len(teams) == 1 && teams[0] == "" {
					teams = []string{}
				}
			}
			ntop = r.MustStr("ntop")
		}

		stakingRequirement := ""
		notionalRequirement := ""
		if r.HasColumn("staking_requirement") {
			stakingRequirement = r.MustStr("staking_requirement")
		}
		if r.HasColumn("notional_requirement") {
			notionalRequirement = r.mustColumn("notional_requirement")
		}

		dispatchStrategy = &proto.DispatchStrategy{
			AssetForMetric:       r.MustStr("metric_asset"),
			Markets:              mkts,
			Metric:               proto.DispatchMetric(proto.DispatchMetric_value[r.MustStr("metric")]),
			DistributionStrategy: distributionStrategy,
			LockPeriod:           lockPeriod,
			EntityScope:          entityScope,
			IndividualScope:      indiScope,
			WindowLength:         windowLength,
			TeamScope:            teams,
			NTopPerformers:       ntop,
			StakingRequirement:   stakingRequirement,
			NotionalTimeWeightedAveragePositionRequirement: notionalRequirement,
			RankTable: ranks,
		}
	}

	factor := num.MustDecimalFromString(r.MustStr("factor"))
	recurring := &types.RecurringTransfer{
		TransferBase: &types.TransferBase{
			ID:              id,
			From:            from,
			FromAccountType: types.AccountType(fromAT),
			To:              to,
			ToAccountType:   types.AccountType(toAT),
			Asset:           asset,
			Amount:          amountUint,
		},
		StartEpoch:       startEpoch.Uint64(),
		EndEpoch:         endEpochPtr,
		Factor:           factor,
		DispatchStrategy: dispatchStrategy,
	}
	return recurring
}

func PartiesCancelTransfers(
	engine *banking.Engine,
	table *godog.Table,
) error {
	errs := []error{}
	for _, r := range parseOneOffCancellationTable(table) {
		err := engine.CancelTransferFunds(context.Background(), &types.CancelTransferFunds{
			Party:      r.MustStr("party"),
			TransferID: r.MustStr("transfer_id"),
		})
		if len(r.Str("error")) > 0 || err != nil {
			expected := r.Str("error")
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if expected != actual {
				errs = append(errs, errors.New(r.MustStr("transfer_id")+" expected "+expected+" but got "+actual))
			}
		}
	}
	if len(errs) > 0 {
		return ErrStack(errs)
	}
	return nil
}

func parseOneOffCancellationTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party", "transfer_id",
	}, []string{"error"})
}
