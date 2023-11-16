package amm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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

type Pool struct {
	ID         string
	SubAccount string
	Commitment *num.Uint
	Parameters *types.ConcentratedLiquidityParameters
}

type Engine struct {
	log *logging.Logger

	broker Broker

	collateral Collateral
	market     Market

	// map of party -> pool
	pools map[string]*Pool
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
) *Engine {
	return &Engine{
		log:                  log,
		broker:               broker,
		collateral:           collateral,
		market:               market,
		pools:                map[string]*Pool{},
		subAccounts:          map[string]string{},
		minCommitmentQuantum: num.UintZero(),
	}
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

	e.pools[submit.Party] = &Pool{
		ID:         deterministicID,
		SubAccount: subAccount,
		Commitment: submit.CommitmentAmount,
		Parameters: submit.Parameters,
	}

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

	pool.Commitment = amend.CommitmentAmount
	pool.Parameters.ApplyUpdate(amend.Parameters)

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
