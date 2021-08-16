package rewards

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

const (
	stakingAndDelegationSchemeID = "staking & delegation"
)

var (
	//ErrUnknownSchemeID is returned when trying to update a reward scheme that isn't already registered
	ErrUnknownSchemeID = errors.New("unknown scheme identifier for update scheme")
	//ErrUnsupported is returned when trying to register a reward scheme - this is not currently supported externally
	ErrUnsupported = errors.New("registering a reward scheme is unsupported")
)

//Broker for sending events
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//EpochEngine notifies the reward engine at the end of an epoch
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

//Delegation engine for getting validation data
//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_engine_mock.go -package mocks code.vegaprotocol.io/vega/rewards DelegationEngine
type Delegation interface {
	OnEpochEnd(ctx context.Context, start, end time.Time) []*types.ValidatorData
}

//Collateral engine provides access to account data and transferring rewards
type Collateral interface {
	CreateOrGetAssetRewardPoolAccount(ctx context.Context, asset string) (string, error)
	GetAccountByID(id string) (*types.Account, error)
	GetPartyGeneralAccount(partyID, asset string) (*types.Account, error)
	TransferRewards(ctx context.Context, rewardAccountID string, transfers []*types.Transfer) ([]*types.TransferResponse, error)
}

//TimeService notifies the reward engine on time updates
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/rewards TimeService
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
	GetTimeNow() time.Time
}

//Engine is the reward engine handling reward payouts
type Engine struct {
	log                              *logging.Logger
	config                           Config
	broker                           Broker
	delegation                       Delegation
	collateral                       Collateral
	rewardSchemes                    map[string]*types.RewardScheme
	pendingPayouts                   map[time.Time][]*pendingPayout
	rewardPoolToPendingPayoutBalance map[string]*num.Uint

	assetForStakingAndDelegationReward string
}

type pendingPayout struct {
	fromAccount   string
	asset         string
	partyToAmount map[string]*num.Uint
	totalReward   *num.Uint
	epochSeq      string
}

//New instantiate a new rewards engine
func New(log *logging.Logger, config Config, broker Broker, delegation Delegation, epochEngine EpochEngine, collateral Collateral, ts TimeService) *Engine {
	e := &Engine{
		config:                           config,
		log:                              log.Named(namedLogger),
		broker:                           broker,
		delegation:                       delegation,
		collateral:                       collateral,
		rewardSchemes:                    map[string]*types.RewardScheme{},
		pendingPayouts:                   map[time.Time][]*pendingPayout{},
		rewardPoolToPendingPayoutBalance: map[string]*num.Uint{},
	}

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEnd)

	// register for time tick updates
	ts.NotifyOnTick(e.onChainTimeUpdate)

	// hack for sweetwater - hardcode reward scheme for staking and delegation
	e.registerStakingAndDelegationRewardScheme()

	return e
}

// this is a hack for sweetwater to hardcode the registeration of reward scheme for staking and delegation in a network scope param.
// so that its parameters can be easily changed they are defined as network params
func (e *Engine) registerStakingAndDelegationRewardScheme() {
	// setup the reward scheme for staking and delegation
	rs := &types.RewardScheme{
		SchemeID:                  stakingAndDelegationSchemeID,
		Type:                      types.RewardSchemeStakingAndDelegation,
		ScopeType:                 types.RewardSchemeScopeNetwork,
		Parameters:                map[string]types.RewardSchemeParam{},
		StartTime:                 time.Now(),
		PayoutType:                types.PayoutFractional,
		MaxPayoutPerAssetPerParty: map[string]*num.Uint{},
	}

	e.rewardSchemes[rs.SchemeID] = rs
}

//UpdateAssetForStakingAndDelegationRewardScheme is called when the asset for staking and delegation is available, get the reward pool account and attach it to the scheme
func (e *Engine) UpdateAssetForStakingAndDelegationRewardScheme(ctx context.Context, asset string) error {
	rs, ok := e.rewardSchemes[stakingAndDelegationSchemeID]
	if !ok {
		e.log.Panic("reward scheme for staking and delegation must exist")
	}

	prevAssetName := e.assetForStakingAndDelegationReward
	e.assetForStakingAndDelegationReward = asset
	rewardAccountID, err := e.collateral.CreateOrGetAssetRewardPoolAccount(ctx, asset)
	if err != nil {
		e.log.Panic("failed to create or get reward account for staking and delegation")
	}
	rs.RewardPoolAccountIDs = []string{rewardAccountID}

	// if the asset comes after the max payout per asset we need to update both
	maxPayout, ok := rs.MaxPayoutPerAssetPerParty[prevAssetName]
	if ok {
		rs.MaxPayoutPerAssetPerParty = map[string]*num.Uint{
			e.assetForStakingAndDelegationReward: maxPayout,
		}
	}
	return nil
}

//UpdateMaxPayoutPerParticipantForStakingRewardScheme is a callback for changes in the network param for max payout per participant
func (e *Engine) UpdateMaxPayoutPerParticipantForStakingRewardScheme(ctx context.Context, mayPayoutPerParticipant uint64) error {
	rs, ok := e.rewardSchemes[stakingAndDelegationSchemeID]
	if !ok {
		e.log.Panic("reward scheme for staking and delegation must exist")
	}

	rs.MaxPayoutPerAssetPerParty[e.assetForStakingAndDelegationReward] = num.NewUint(mayPayoutPerParticipant)
	return nil
}

//UpdatePayoutFractionForStakingRewardScheme is a callback for changes in the network param for payout fraction
func (e *Engine) UpdatePayoutFractionForStakingRewardScheme(ctx context.Context, payoutFraction float64) error {
	rs, ok := e.rewardSchemes[stakingAndDelegationSchemeID]
	if !ok {
		e.log.Panic("reward scheme for staking and delegation must exist")
	}
	rs.PayoutFraction = payoutFraction
	return nil
}

//UpdatePayoutDelayForStakingRewardScheme is a callback for changes in the network param for payout delay
func (e *Engine) UpdatePayoutDelayForStakingRewardScheme(ctx context.Context, payoutDelay time.Duration) error {
	rs, ok := e.rewardSchemes[stakingAndDelegationSchemeID]
	if !ok {
		e.log.Panic("reward scheme for staking and delegation must exist")
	}
	rs.PayoutDelay = payoutDelay
	return nil
}

//UpdateDelegatorShareForStakingRewardScheme is a callback for changes in the network param for delegator share
func (e *Engine) UpdateDelegatorShareForStakingRewardScheme(ctx context.Context, delegatorShare float64) error {
	rs, ok := e.rewardSchemes[stakingAndDelegationSchemeID]
	if !ok {
		e.log.Panic("reward scheme for staking and delegation must exist")
	}
	rs.Parameters["delegatorShare"] = types.RewardSchemeParam{
		Name:  "delegatorShare",
		Type:  "float",
		Value: fmt.Sprintf("%f", delegatorShare),
	}
	return nil
}

//RegisterRewardScheme allows registration of a new reward scheme - unsupported for now
func (e *Engine) RegisterRewardScheme(rs *types.RewardScheme) error {
	return ErrUnsupported
}

//UpdateRewardScheme updates an existing reward scheme - unsupported for now
func (e *Engine) UpdateRewardScheme(rs *types.RewardScheme) error {
	return ErrUnsupported
}

//whenever we have a time update, check if there are pending payouts ready to be sent
func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	payTimes := make([]time.Time, 0, len(e.pendingPayouts))
	for payTime := range e.pendingPayouts {
		payTimes = append(payTimes, payTime)
	}
	sort.Slice(payTimes, func(i, j int) bool { return payTimes[i].Before(payTimes[j]) })
	for _, payTime := range payTimes {
		payouts := e.pendingPayouts[payTime]
		if t.After(payTime) {
			for _, payout := range payouts {
				// distribute the reward
				if payout != nil {
					e.distributePayout(ctx, payout)
				}

				// subtract the reward from the pending balance
				pendingBalanceForRewardAccount := e.rewardPoolToPendingPayoutBalance[payout.fromAccount]
				e.rewardPoolToPendingPayoutBalance[payout.fromAccount] = num.Zero().Sub(pendingBalanceForRewardAccount, payout.totalReward)
			}
			// remove all paid payouts from pending
			delete(e.pendingPayouts, payTime)
		}
	}
}

// OnEpochEnd calculates the reward amounts parties get for available reward schemes
func (e *Engine) OnEpochEnd(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("OnEpochEnd")

	rsIDs := make([]string, 0, len(e.rewardSchemes))
	for rsID := range e.rewardSchemes {
		rsIDs = append(rsIDs, rsID)
	}
	sort.Strings(rsIDs)
	for _, rsID := range rsIDs {
		rewardScheme := e.rewardSchemes[rsID]

		// if reward scheme is not active yet or anymore, ignore it
		if !rewardScheme.IsActive(epoch.EndTime) {
			continue
		}

		// get the reward pool accounts for the reward scheme
		for _, accountID := range rewardScheme.RewardPoolAccountIDs {
			account, err := e.collateral.GetAccountByID(accountID)
			if err != nil {
				e.log.Error("failed to get reward account for", logging.String("accountID", accountID))
				continue
			}

			if account.Balance.IsZero() {
				e.log.Debug("reward account has zero balance", logging.String("accountID", accountID))
				continue
			}

			rewardAccountBalance := account.Balance.Clone()

			// we need to subtract from the balance any pending payouts that are waiting to be awarded
			pendingPayoutForAccount, ok := e.rewardPoolToPendingPayoutBalance[accountID]
			if ok {
				if pendingPayoutForAccount.GT(rewardAccountBalance) {
					e.log.Panic("reward account balance doesn't cover pending payouts")
				}
				rewardAccountBalance = rewardAccountBalance.Sub(rewardAccountBalance, pendingPayoutForAccount)
			} else {
				pendingPayoutForAccount = num.Zero()
			}

			if rewardAccountBalance.IsZero() {
				e.log.Debug("reward account has zero balance including pending payouts", logging.String("accountID", accountID))
				continue
			}

			// get how much reward needs to be distributed based on the current balance and the reward scheme
			rewardAmt, err := rewardScheme.GetReward(rewardAccountBalance, epoch)
			if err != nil {
				e.log.Panic("reward scheme misconfiguration", logging.Error(err))
			}

			// calculate the rewards per the reward scheme and reword amount
			pending := e.calculateRewards(ctx, account.Asset, account.ID, rewardScheme, rewardAmt, epoch)
			if pending.totalReward.IsZero() {
				continue
			}

			// if the reward scheme has no delay, distribute the payout now
			if rewardScheme.PayoutDelay == time.Duration(0) {
				e.distributePayout(ctx, pending)
				continue
			}

			// add the total reward amount to the pending for the account so we can account for it when distributing further rewards
			// if we need to before this is paid out
			e.rewardPoolToPendingPayoutBalance[accountID] = pendingPayoutForAccount.AddSum(pending.totalReward)
			timeToSend := epoch.EndTime.Add(rewardScheme.PayoutDelay)
			existingPending, ok := e.pendingPayouts[timeToSend]
			if !ok {
				existingPending = []*pendingPayout{}
			}
			existingPending = append(existingPending, pending)
			e.pendingPayouts[timeToSend] = existingPending

		}
	}
}

// make the required transfers for distributing reward payout
func (e *Engine) distributePayout(ctx context.Context, payout *pendingPayout) {
	partyAccountIDToParty := make(map[string]string, len(payout.partyToAmount))
	partyIDs := make([]string, 0, len(payout.partyToAmount))
	for party := range payout.partyToAmount {
		partyIDs = append(partyIDs, party)
	}

	sort.Strings(partyIDs)
	transfers := make([]*types.Transfer, 0, len(partyIDs))
	for _, party := range partyIDs {
		amt := payout.partyToAmount[party]
		transfers = append(transfers, &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Asset:  payout.asset,
				Amount: amt.Clone(),
			},
			Type:      types.TransferTypeRewardPayout,
			MinAmount: amt.Clone(),
		})

	}

	resp, err := e.collateral.TransferRewards(ctx, payout.fromAccount, transfers)
	if err != nil {
		e.log.Error("error in transfer rewards", logging.Error(err))
		return
	}

	// emit events
	payoutEvents := map[string]*events.RewardPayout{}
	parties := []string{}
	for _, response := range resp {
		// send an event with the reward amount transferred to the party
		if len(response.Transfers) > 0 {
			ledgerEntry := response.Transfers[0]
			party := partyAccountIDToParty[ledgerEntry.ToAccount]
			payoutEvents[party] = events.NewRewardPayout(ctx, ledgerEntry.FromAccount, ledgerEntry.ToAccount, party, payout.epochSeq, payout.asset, ledgerEntry.Amount, ledgerEntry.Amount.Float64()/payout.totalReward.Float64())
			parties = append(parties, party)
		}
	}
	sort.Strings(parties)
	payoutEventSlice := make([]events.Event, 0, len(parties))
	for _, p := range parties {
		payoutEventSlice = append(payoutEventSlice, *payoutEvents[p])
	}
	e.broker.SendBatch(payoutEventSlice)
}

// delegates the reward calculation to the reward scheme
//NB currently the only reward scheme type supported is staking and delegation
func (e *Engine) calculateRewards(ctx context.Context, asset string, accountID string, rewardScheme *types.RewardScheme, rewardBalance *num.Uint, epoch types.Epoch) *pendingPayout {
	if rewardScheme.Type != types.RewardSchemeStakingAndDelegation {
		e.log.Panic("unsupported reward scheme type", logging.Int("type", int(rewardScheme.Type)))
	}

	// get the validator delegation data from the delegation engine and calculate the staking and delegation rewards for the epoch
	validatorData := e.delegation.OnEpochEnd(ctx, epoch.StartTime, epoch.EndTime)
	return e.calculatStakingAndDelegationRewards(asset, accountID, rewardScheme, rewardBalance, validatorData)
}
