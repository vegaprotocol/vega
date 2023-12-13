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

package amm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var (
	ErrNoPoolMatchingParty  = errors.New("no pool matching party")
	ErrPartyAlreadyOwnAPool = func(market string) error {
		return fmt.Errorf("party already own a pool for market %v", market)
	}
	ErrCommitmentTooLow          = errors.New("commitment amount too low")
	ErrRebaseOrderDidNotTrade    = errors.New("rebase-order did not trade")
	ErrRebaseTargetOutsideBounds = errors.New("rebase target outside bounds")
)

const (
	version = "AMMv1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/amm Collateral,Position,Market,Risk

type Collateral interface {
	GetAssetQuantum(asset string) (num.Decimal, error)
	GetPartyMarginAccount(market, party, asset string) (*types.Account, error)
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
	SubAccountUpdate(
		ctx context.Context,
		party, subAccount, asset, market string,
		transferType types.TransferType,
		amount *num.Uint,
	) (*types.LedgerMovement, error)
	CreatePartyAMMsSubAccounts(
		ctx context.Context,
		party, subAccount, asset, market string,
	) (general *types.Account, margin *types.Account, err error)
}

type Broker interface {
	Send(events.Event)
}

type Market interface {
	GetID() string
	ClosePosition(context.Context, string) bool // return true if position was successfully closed
	GetSettlementAsset() string
	SubmitOrderWithIDGeneratorAndOrderID(context.Context, *types.OrderSubmission, string, common.IDGenerator, string, bool) (*types.OrderConfirmation, error)
}

type Risk interface {
	GetRiskFactors() *types.RiskFactor
	GetScalingFactors() *types.ScalingFactors
	GetSlippage() num.Decimal
}

type Position interface {
	GetPositionsByParty(ids ...string) []events.MarketPosition
}

type sqrtFn func(*num.Uint) *num.Uint

// Sqrter calculates sqrt's of Uints and caches the results. We want this cache to be shared across all pools for a market.
type Sqrter struct {
	cache map[string]*num.Uint
}

// sqrt calculates the square root of the uint and caches it.
func (s *Sqrter) sqrt(u *num.Uint) *num.Uint {
	if r := s.cache[u.String()]; r != nil {
		return r.Clone()
	}

	// for now lets just use the sqrt algo in the uint256 library and if its slow
	// we can work something out later
	r := num.UintOne().Sqrt(u)

	// we can also maybe be more clever here and use a LRU but whatever
	s.cache[u.String()] = r
	return r.Clone()
}

type Engine struct {
	log *logging.Logger

	broker Broker

	risk       Risk
	collateral Collateral
	position   Position
	market     Market

	// gets us from the price in the submission -> price in full asset dp
	priceFactor *num.Uint

	// map of party -> pool
	pools map[string]*Pool

	// sqrt calculator with cache
	rooter *Sqrter

	// a mapping of all sub accounts to the party owning them.
	subAccounts map[string]string

	minCommitmentQuantum *num.Uint
}

func New(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	risk Risk,
	position Position,
	priceFactor *num.Uint,
) *Engine {
	return &Engine{
		log:                  log,
		broker:               broker,
		risk:                 risk,
		collateral:           collateral,
		position:             position,
		market:               market,
		pools:                map[string]*Pool{},
		subAccounts:          map[string]string{},
		minCommitmentQuantum: num.UintZero(),
		rooter:               &Sqrter{cache: map[string]*num.Uint{}},
		priceFactor:          priceFactor,
	}
}

func NewFromProto(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	risk Risk,
	position Position,
	state *v1.AmmState,
	priceFactor *num.Uint,
) *Engine {
	e := New(log, broker, collateral, market, risk, position, priceFactor)

	for _, v := range state.SubAccounts {
		e.subAccounts[v.Key] = v.Value
	}

	for _, v := range state.Sqrter {
		e.rooter.cache[v.Key] = num.MustUintFromString(v.Value, 10)
	}

	for _, v := range state.Pools {
		e.pools[v.Party] = NewPoolFromProto(e.rooter.sqrt, e.collateral, e.position, v.Pool)
	}

	return e
}

func (e *Engine) IntoProto() *v1.AmmState {
	state := &v1.AmmState{
		Sqrter:      make([]*v1.StringMapEntry, 0, len(e.rooter.cache)),
		SubAccounts: make([]*v1.StringMapEntry, 0, len(e.subAccounts)),
		Pools:       make([]*v1.PoolMapEntry, 0, len(e.pools)),
	}

	for k, v := range e.rooter.cache {
		state.Sqrter = append(state.Sqrter, &v1.StringMapEntry{
			Key:   k,
			Value: v.String(),
		})
	}
	sort.Slice(state.Sqrter, func(i, j int) bool { return state.Sqrter[i].Key < state.Sqrter[j].Key })

	for k, v := range e.subAccounts {
		state.SubAccounts = append(state.SubAccounts, &v1.StringMapEntry{
			Key:   k,
			Value: v,
		})
	}
	sort.Slice(state.SubAccounts, func(i, j int) bool { return state.SubAccounts[i].Key < state.SubAccounts[j].Key })

	for k, v := range e.pools {
		state.Pools = append(state.Pools, &v1.PoolMapEntry{
			Party: k,
			Pool:  v.IntoProto(),
		})
	}
	sort.Slice(state.Pools, func(i, j int) bool { return state.Pools[i].Party < state.Pools[j].Party })

	return state
}

func (e *Engine) OnMinCommitmentQuantumUpdate(ctx context.Context, c *num.Uint) {
	e.minCommitmentQuantum = c.Clone()
}

// TBD.
func (e *Engine) OnTick(ctx context.Context, _ time.Time) {
	// check sub account balances (margin, general)
}

func (e *Engine) IsPoolSubAccount(key string) bool {
	_, yes := e.subAccounts[key]
	return yes
}

func (e *Engine) SubmitAMM(
	ctx context.Context,
	submit *types.SubmitAMM,
	deterministicID string,
	targetPrice *num.Uint,
) error {
	subAccount := DeriveSubAccount(submit.Party, submit.MarketID, version, 0)
	_, ok := e.pools[submit.Party]
	if ok {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonPartyAlreadyOwnsAPool,
			),
		)

		return ErrPartyAlreadyOwnAPool(e.market.GetID())
	}

	if err := e.ensureCommitmentAmount(ctx, submit.CommitmentAmount); err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCommitmentTooLow,
			),
		)
		return err
	}

	_, _, err := e.collateral.CreatePartyAMMsSubAccounts(ctx, submit.Party, subAccount, e.market.GetSettlementAsset(), submit.MarketID)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonUnspecified,
			),
		)

		return err
	}

	err = e.updateSubAccountBalance(
		ctx, submit.Party, subAccount, submit.CommitmentAmount,
	)
	if err != nil {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCannotFillCommitment,
			),
		)

		return err
	}

	pool := NewPool(
		deterministicID,
		subAccount,
		e.market.GetSettlementAsset(),
		submit,
		e.rooter.sqrt,
		e.collateral,
		e.position,
		e.risk.GetRiskFactors(),
		e.risk.GetScalingFactors(),
		e.risk.GetSlippage(),
		e.priceFactor,
	)

	if targetPrice != nil {
		if err := e.rebasePool(ctx, pool, targetPrice, submit.SlippageTolerance); err != nil {
			if err := e.updateSubAccountBalance(ctx, submit.Party, subAccount, num.UintZero()); err != nil {
				e.log.Panic("unable to remove sub account balances", logging.Error(err))
			}

			// couldn't rebase the pool so it gets rejected
			e.broker.Send(
				events.NewAMMPoolEvent(
					ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
					submit.CommitmentAmount, submit.Parameters,
					types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCannotRebase,
				),
			)
			return err
		}
	}

	e.pools[submit.Party] = pool
	e.broker.Send(
		events.NewAMMPoolEvent(
			ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
			submit.CommitmentAmount, submit.Parameters,
			types.AMMPoolStatusActive, types.AMMPoolStatusReasonUnspecified,
		),
	)
	return nil
}

func (e *Engine) AmendAMM(
	ctx context.Context,
	amend *types.AmendAMM,
) error {
	pool, ok := e.pools[amend.Party]
	if !ok {
		return ErrNoPoolMatchingParty
	}

	if err := e.ensureCommitmentAmount(ctx, amend.CommitmentAmount); err != nil {
		return err
	}

	fairPrice := pool.TradePrice(&types.Order{})
	oldCommitment := pool.Commitment.Clone()

	err := e.updateSubAccountBalance(
		ctx, amend.Party, pool.SubAccount, amend.CommitmentAmount,
	)
	if err != nil {
		return err
	}

	pool.Update(amend, e.risk.GetRiskFactors(), e.risk.GetScalingFactors(), e.risk.GetSlippage())
	if err := e.rebasePool(ctx, pool, fairPrice, amend.SlippageTolerance); err != nil {
		// couldn't rebase the pool back to its original fair price so the amend is rejected
		if err := e.updateSubAccountBalance(ctx, amend.Party, pool.SubAccount, oldCommitment); err != nil {
			e.log.Panic("could not revert balances are failed rebase", logging.Error(err))
		}
		return err
	}

	e.broker.Send(
		events.NewAMMPoolEvent(
			ctx, amend.Party, e.market.GetID(), pool.SubAccount, pool.ID,
			amend.CommitmentAmount, amend.Parameters,
			types.AMMPoolStatusActive, types.AMMPoolStatusReasonUnspecified,
		),
	)
	return nil
}

func (e *Engine) CancelAMM(
	ctx context.Context,
	cancel *types.CancelAMM,
) error {
	pool, ok := e.pools[cancel.Party]
	if !ok {
		return ErrNoPoolMatchingParty
	}

	err := e.updateSubAccountBalance(
		ctx, cancel.Party, pool.SubAccount, num.UintZero(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (e *Engine) StopPool(
	_ context.Context,
	key string,
) error {
	party, ok := e.subAccounts[key]
	if !ok {
		return ErrNoPoolMatchingParty
	}

	_ = party

	return nil
}

func (e *Engine) MarketClosing() error { return errors.New("unimplemented") }

func (e *Engine) ensureCommitmentAmount(
	_ context.Context,
	commitmentAmount *num.Uint,
) error {
	quantum, _ := e.collateral.GetAssetQuantum(e.market.GetSettlementAsset())
	quantumCommitment := commitmentAmount.ToDecimal().Div(quantum)

	if quantumCommitment.LessThan(e.minCommitmentQuantum.ToDecimal()) {
		return ErrCommitmentTooLow
	}

	return nil
}

func (e *Engine) updateSubAccountBalance(
	ctx context.Context,
	party, subAccount string,
	newCommitment *num.Uint,
) error {
	// first we get the current balance of both the margin, and general subAccount
	subMargin, err := e.collateral.GetPartyMarginAccount(
		e.market.GetID(), subAccount, e.market.GetSettlementAsset())
	if err != nil {
		// by that point the account must exist
		e.log.Panic("no sub margin account", logging.Error(err))
	}
	subGeneral, err := e.collateral.GetPartyGeneralAccount(
		subAccount, e.market.GetSettlementAsset())
	if err != nil {
		// by that point the account must exist
		e.log.Panic("no sub general account", logging.Error(err))
	}

	var (
		currentCommitment = num.Sum(subMargin.Balance, subGeneral.Balance)
		transferType      types.TransferType
		actualAmount      = num.UintZero()
	)

	if currentCommitment.LT(newCommitment) {
		transferType = types.TransferTypeAMMSubAccountLow
		actualAmount.Sub(newCommitment, currentCommitment)
	} else if currentCommitment.GT(newCommitment) {
		transferType = types.TransferTypeAMMSubAcountHigh
		actualAmount.Sub(currentCommitment, newCommitment)
	} else {
		// nothing to do
		return nil
	}

	ledgerMovements, err := e.collateral.SubAccountUpdate(
		ctx, party, subAccount, e.market.GetSettlementAsset(),
		e.market.GetID(), transferType, actualAmount,
	)
	if err != nil {
		return err
	}

	e.broker.Send(events.NewLedgerMovements(
		ctx, []*types.LedgerMovement{ledgerMovements}))

	return nil
}

// rebasePool submits an order on behalf of the given pool to pull it fair-price towards the target.
func (e *Engine) rebasePool(ctx context.Context, pool *Pool, target *num.Uint, tol num.Decimal) error {
	if target.LT(pool.lower.low) || target.GT(pool.upper.high) {
		return ErrRebaseTargetOutsideBounds
	}

	// get the pools current fair-price
	fairPrice := pool.TradePrice(&types.Order{})
	e.log.Debug("rebasing pool",
		logging.String("id", pool.ID),
		logging.String("fair-price", fairPrice.String()),
		logging.String("target", target.String()),
		logging.String("slippage", tol.String()),
	)
	if fairPrice.EQ(target) {
		return nil
	}

	// calculate slippage as a factor of the mark-price so we can allow for a trades at prices +/- either side of the mark price, depending on side
	slippage, _ := num.UintFromDecimal(target.ToDecimal().Mul(tol))

	// this is the order the pool will submit to rebase itself such that its fair-price is roughly the mark price
	order := &types.OrderSubmission{
		MarketID:    pool.market,
		Price:       num.UintZero(),
		TimeInForce: types.OrderTimeInForceFOK,
		Type:        types.OrderTypeLimit,
		Reference:   fmt.Sprintf("amm-rebase-%s", pool.ID),
	}

	if target.GT(fairPrice) {
		// pool base price is higher than market price, it will need to sell to lower its fair-price
		order.Side = types.SideSell
		order.Price.Sub(target, slippage)
	} else {
		order.Side = types.SideBuy
		order.Price.Add(target, slippage)
	}

	// ask the pool for the volume it would need to shift to get its price to target
	order.Size = pool.VolumeBetweenPrices(order.Side, fairPrice, target)
	if order.Size == 0 {
		// fair-price is so close to target price that the volume to shift it is too small, but thats ok
		return nil
	}

	// need to scale make to market precision here because thats what SubmitOrderWithIDGeneratorAndOrderID expects
	order.Price.Div(order.Price, e.priceFactor)

	e.log.Debug("submitting order to rebase after scale",
		logging.Uint64("size", order.Size),
		logging.String("price", order.Price.String()),
		logging.String("side", order.Side.String()),
	)
	idgen := idgeneration.New(pool.ID)

	conf, err := e.market.SubmitOrderWithIDGeneratorAndOrderID(ctx, order, pool.SubAccount, idgen, idgen.NextID(), true)
	if err != nil {
		return err
	}

	if conf.Order.Status != types.OrderStatusFilled {
		return ErrRebaseOrderDidNotTrade
	}
	return nil
}

func DeriveSubAccount(
	party, market, version string,
	index uint64,
) string {
	hash := crypto.Hash([]byte(fmt.Sprintf("%v%v%v%v", version, market, party, index)))
	return hex.EncodeToString(hash)
}
