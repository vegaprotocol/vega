package amm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
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
	ErrCommitmentTooLow = errors.New("commitment amount too low")
)

const (
	version = "AMMv1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/amm Collateral,Position
type Collateral interface {
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

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type Market interface {
	GetID() string
	ClosePosition(context.Context, string) bool // return true if position was succesfully closed
	GetSettlementAsset() string
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

	// map of party -> pool
	pools map[string]*Pool

	// sqrt calculator with cache
	rooter *Sqrter

	// a mapping of all sub accounts to the party owning them.
	subAccounts map[string]string

	minCommitmentQuantum *num.Uint

	assets Assets
}

func New(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	assets Assets,
	risk Risk,
	position Position,
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
	}
}

func NewFromProto(
	log *logging.Logger,
	broker Broker,
	collateral Collateral,
	market Market,
	assets Assets,
	risk Risk,
	position Position,
	state *v1.AmmState,
) *Engine {
	e := New(log, broker, collateral, market, assets, risk, position)

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

// TBD
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
) error {
	subAccount := DeriveSubAccount(submit.Party, submit.MarketID, version, 0)
	_, ok := e.pools[submit.Party]
	if ok {
		e.broker.Send(
			events.NewAMMPoolEvent(
				ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
				submit.CommitmentAmount, submit.Parameters,
				types.AMMPoolStatusRejected, types.AMMPoolStatusReasonPartyAlreadyOwnAPool,
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

	err := e.updateSubAccountBalance(
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
	)

	e.pools[submit.Party] = pool
	events.NewAMMPoolEvent(
		ctx, submit.Party, e.market.GetID(), subAccount, deterministicID,
		submit.CommitmentAmount, submit.Parameters,
		types.AMMPoolStatusActive, types.AMMPoolStatusReasonUnspecified,
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

	err := e.updateSubAccountBalance(
		ctx, amend.Party, pool.SubAccount, amend.CommitmentAmount,
	)
	if err != nil {
		return err
	}

	pool.Update(amend, e.risk.GetRiskFactors(), e.risk.GetScalingFactors(), e.risk.GetSlippage())

	e.broker.Send(
		events.NewAMMPoolEvent(
			ctx, amend.Party, e.market.GetID(), pool.SubAccount, pool.ID,
			amend.CommitmentAmount, amend.Parameters,
			types.AMMPoolStatusRejected, types.AMMPoolStatusReasonCannotFillCommitment,
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
	ctx context.Context,
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
	ctx context.Context,
	commitmentAmount *num.Uint,
) error {
	asset, _ := e.assets.Get(e.market.GetSettlementAsset())
	quantum := asset.Type().Details.Quantum
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

func DeriveSubAccount(
	party, market, version string,
	index uint64,
) string {
	hash := crypto.Hash([]byte(fmt.Sprintf("%v%v%v%v", version, market, party, index)))
	return hex.EncodeToString(hash)
}
