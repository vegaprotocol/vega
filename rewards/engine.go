package rewards

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

const (
	stakingAndDelegationSchemeID = "staking & delegation"
	infrstructureFeeSchemeID     = "infrastructure fee"
)

var (
	// ErrUnknownSchemeID is returned when trying to update a reward scheme that isn't already registered.
	ErrUnknownSchemeID = errors.New("unknown scheme identifier for update scheme")
	// ErrUnsupported is returned when trying to register a reward scheme - this is not currently supported externally.
	ErrUnsupported = errors.New("registering a reward scheme is unsupported")

	votingPowerScalingFactor, _ = num.DecimalFromString("10000")
	decimal1, _                 = num.DecimalFromString("1")
)

// Broker for sending events.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// EpochEngine notifies the reward engine at the end of an epoch.
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
}

//Delegation engine for getting validation data
//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_engine_mock.go -package mocks code.vegaprotocol.io/vega/rewards Delegation
type Delegation interface {
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	GetValidatorData() []*types.ValidatorData
}

// Collateral engine provides access to account data and transferring rewards.
type Collateral interface {
	CreateOrGetAssetRewardPoolAccount(ctx context.Context, asset string) (string, error)
	GetAccountByID(id string) (*types.Account, error)
	TransferRewards(ctx context.Context, rewardAccountID string, transfers []*types.Transfer) ([]*types.TransferResponse, error)
	GetInfraFeeAccountIDs() []string
}

//TimeService notifies the reward engine on time updates
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/rewards TimeService
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
	GetTimeNow() time.Time
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/val_performance_mock.go -package mocks code.vegaprotocol.io/vega/rewards ValidatorPerformance
type ValidatorPerformance interface {
	ValidatorPerformanceScore(address string) num.Decimal
}

// Engine is the reward engine handling reward payouts.
type Engine struct {
	log                                *logging.Logger
	config                             Config
	broker                             Broker
	delegation                         Delegation
	collateral                         Collateral
	valPerformance                     ValidatorPerformance
	rewardSchemes                      map[string]*types.RewardScheme // reward scheme id -> reward scheme
	pendingPayouts                     map[time.Time][]*payout
	assetForStakingAndDelegationReward string
	rss                                *rewardsSnapshotState
	rng                                *rand.Rand
	global                             *globalRewardParams
	newEpochStarted                    bool // flag to signal new epoch so we can update the voting power at the end of the block
	epochSeq                           string
}

type globalRewardParams struct {
	maxPerEpoch             *num.Uint
	minValStakeD            num.Decimal
	minValStakeUInt         *num.Uint
	optimalStakeMultiplier  num.Decimal
	compLevel               num.Decimal
	minValidators           num.Decimal
	maxPayoutPerParticipant *num.Uint
	payoutDelay             time.Duration
	delegatorShare          num.Decimal
}

type payout struct {
	fromAccount   string
	asset         string
	partyToAmount map[string]*num.Uint
	totalReward   *num.Uint
	epochSeq      string
	timestamp     int64
}

// New instantiate a new rewards engine.
func New(log *logging.Logger, config Config, broker Broker, delegation Delegation, epochEngine EpochEngine, collateral Collateral, ts TimeService, valPerformance ValidatorPerformance) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:         config,
		log:            log.Named(namedLogger),
		broker:         broker,
		delegation:     delegation,
		collateral:     collateral,
		rewardSchemes:  map[string]*types.RewardScheme{},
		pendingPayouts: map[time.Time][]*payout{},
		rss: &rewardsSnapshotState{
			changed:    true,
			hash:       []byte{},
			serialised: []byte{},
		},
		global:          &globalRewardParams{},
		newEpochStarted: false,
		valPerformance:  valPerformance,
	}

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent)

	// register for time tick updates
	ts.NotifyOnTick(e.onChainTimeUpdate)

	// hack for sweetwater - hardcode reward scheme for staking and delegation
	e.registerStakingAndDelegationRewardScheme()

	// register the infrastructure fee scheme
	e.registerInfrastructureFeeRewardScheme()
	return e
}

// register the infrastructure fee reward scheme.
func (e *Engine) registerInfrastructureFeeRewardScheme() {
	// setup the reward scheme for staking and delegation
	rs := &types.RewardScheme{
		SchemeID:                  infrstructureFeeSchemeID,
		Type:                      types.RewardSchemeInfrastructureFee,
		ScopeType:                 types.RewardSchemeScopeNetwork,
		Parameters:                map[string]types.RewardSchemeParam{},
		StartTime:                 time.Time{},
		PayoutType:                types.PayoutFractional,
		MaxPayoutPerAssetPerParty: map[string]*num.Uint{},
	}

	e.rewardSchemes[rs.SchemeID] = rs
}

// this is a hack for sweetwater to hardcode the registeration of reward scheme for staking and delegation in a network scope param.
// so that its parameters can be easily changed they are defined as network params.
func (e *Engine) registerStakingAndDelegationRewardScheme() {
	// setup the reward scheme for staking and delegation
	rs := &types.RewardScheme{
		SchemeID:                  stakingAndDelegationSchemeID,
		Type:                      types.RewardSchemeStakingAndDelegation,
		ScopeType:                 types.RewardSchemeScopeNetwork,
		Parameters:                map[string]types.RewardSchemeParam{},
		StartTime:                 time.Time{},
		PayoutType:                types.PayoutFractional,
		MaxPayoutPerAssetPerParty: map[string]*num.Uint{},
	}

	e.rewardSchemes[rs.SchemeID] = rs
}

// UpdateMaxPayoutPerEpochStakeForStakingRewardScheme controls the max payout per epoch.
func (e *Engine) UpdateMaxPayoutPerEpochStakeForStakingRewardScheme(ctx context.Context, maxPerEpoch num.Decimal) error {
	e.global.maxPerEpoch, _ = num.UintFromDecimal(maxPerEpoch)
	return nil
}

// UpdateMinimumValidatorStakeForStakingRewardScheme updaates the value of minimum validator stake for being considered for rewards.
func (e *Engine) UpdateMinimumValidatorStakeForStakingRewardScheme(ctx context.Context, minValStake num.Decimal) error {
	e.global.minValStakeD = minValStake
	e.global.minValStakeUInt, _ = num.UintFromDecimal(minValStake)
	return nil
}

// UpdateOptimalStakeMultiplierStakingRewardScheme updaates the value of optimal stake multiplier.
func (e *Engine) UpdateOptimalStakeMultiplierStakingRewardScheme(ctx context.Context, optimalStakeMultiplier num.Decimal) error {
	e.global.optimalStakeMultiplier = optimalStakeMultiplier
	return nil
}

// UpdateCompetitionLevelForStakingRewardScheme is called when the competition level has changed.
func (e *Engine) UpdateCompetitionLevelForStakingRewardScheme(ctx context.Context, compLevel num.Decimal) error {
	e.global.compLevel = compLevel
	return nil
}

// UpdateMinValidatorsStakingRewardScheme is called when the the network parameter for min validator has changed.
func (e *Engine) UpdateMinValidatorsStakingRewardScheme(ctx context.Context, minValidators int64) error {
	e.global.minValidators = num.DecimalFromInt64(minValidators)
	return nil
}

// UpdateAssetForStakingAndDelegationRewardScheme is called when the asset for staking and delegation is available, get the reward pool account and attach it to the scheme.
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

// UpdateMaxPayoutPerParticipantForStakingRewardScheme is a callback for changes in the network param for max payout per participant.
func (e *Engine) UpdateMaxPayoutPerParticipantForStakingRewardScheme(ctx context.Context, maxPayoutPerParticipant num.Decimal) error {
	e.global.maxPayoutPerParticipant, _ = num.UintFromDecimal(maxPayoutPerParticipant)
	return nil
}

// UpdatePayoutFractionForStakingRewardScheme is a callback for changes in the network param for payout fraction.
func (e *Engine) UpdatePayoutFractionForStakingRewardScheme(ctx context.Context, payoutFraction num.Decimal) error {
	for _, rs := range e.rewardSchemes {
		if rs.PayoutType == types.PayoutFractional {
			rs.PayoutFraction = payoutFraction
		}
	}
	return nil
}

// UpdatePayoutDelayForStakingRewardScheme is a callback for changes in the network param for payout delay.
func (e *Engine) UpdatePayoutDelayForStakingRewardScheme(ctx context.Context, payoutDelay time.Duration) error {
	e.global.payoutDelay = payoutDelay
	return nil
}

// UpdateDelegatorShareForStakingRewardScheme is a callback for changes in the network param for delegator share.
func (e *Engine) UpdateDelegatorShareForStakingRewardScheme(ctx context.Context, delegatorShare num.Decimal) error {
	e.global.delegatorShare = delegatorShare
	return nil
}

// RegisterRewardScheme allows registration of a new reward scheme - unsupported for now.
func (e *Engine) RegisterRewardScheme(rs *types.RewardScheme) error {
	return ErrUnsupported
}

// UpdateRewardScheme updates an existing reward scheme - unsupported for now.
func (e *Engine) UpdateRewardScheme(rs *types.RewardScheme) error {
	return ErrUnsupported
}

// whenever we have a time update, check if there are pending payouts ready to be sent.
func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	// resetting the seed every block, to both get some more unpredictability and still deterministic
	// and play nicely with snapshot
	e.rng = rand.New(rand.NewSource(t.Unix()))

	// check if we have any outstanding payouts that need to be distributed
	payTimes := make([]time.Time, 0, len(e.pendingPayouts))
	for payTime := range e.pendingPayouts {
		if !t.Before(payTime) {
			payTimes = append(payTimes, payTime)
		}
	}
	sort.Slice(payTimes, func(i, j int) bool { return payTimes[i].Before(payTimes[j]) })
	for _, payTime := range payTimes {
		// remove all paid payouts from pending
		for _, p := range e.pendingPayouts[payTime] {
			e.distributePayout(ctx, p)
		}
		delete(e.pendingPayouts, payTime)
		e.rss.changed = true
	}
}

func (e *Engine) calcTotalPendingPayout(accountID string) *num.Uint {
	totalPendingForRS := num.Zero()
	for _, payouts := range e.pendingPayouts {
		for _, po := range payouts {
			if po.fromAccount == accountID {
				totalPendingForRS.AddSum(po.totalReward)
			}
		}
	}
	return totalPendingForRS
}

// process rewards when needed.
func (e *Engine) processRewards(ctx context.Context, rewardScheme *types.RewardScheme, epoch types.Epoch, validatorData []*types.ValidatorData, validatorNormalisedScores map[string]num.Decimal, onChainTreasury bool) []*payout {
	payouts := []*payout{}

	// get the reward pool accounts for the reward scheme
	for _, accountID := range rewardScheme.RewardPoolAccountIDs {
		account, err := e.collateral.GetAccountByID(accountID)
		if err != nil {
			e.log.Error("failed to get reward account for", logging.String("accountID", accountID))
			continue
		}

		rewardAccountBalance := account.Balance
		e.log.Info("Rewards: reward account balance for epoch", logging.Uint64("epoch", epoch.Seq), logging.String("rewardAccountBalance", rewardAccountBalance.String()))

		// account for pending payouts
		totalPendingPayouts := e.calcTotalPendingPayout(account.ID)
		if rewardAccountBalance.LT(totalPendingPayouts) {
			e.log.Panic("insufficient balance in reward account to cover for pending payouts", logging.String("rewardAccountBalance", rewardAccountBalance.String()), logging.String("totalPendingPayouts", totalPendingPayouts.String()))
		}
		e.log.Info("Rewards: total pending reward payouts", logging.Uint64("epoch", epoch.Seq), logging.String("totalPendingPayouts", totalPendingPayouts.String()))

		rewardAccountBalance = num.Zero().Sub(rewardAccountBalance, totalPendingPayouts)
		e.log.Info("Rewards: effective reward account balance for epoch", logging.Uint64("epoch", epoch.Seq), logging.String("effectiveRewardBalance", rewardAccountBalance.String()))

		// get how much reward needs to be distributed based on the current balance and the reward scheme
		rewardAmt, err := rewardScheme.GetReward(rewardAccountBalance, epoch)
		if err != nil {
			e.log.Panic("reward scheme misconfiguration", logging.Error(err))
		}

		e.log.Info("Rewards: reward account pot for epoch", logging.Uint64("epoch", epoch.Seq), logging.String("rewardAmt", rewardAmt.String()))

		maxPayoutPerParticipant := num.Zero()
		if onChainTreasury {
			rewardAmt = num.Min(e.global.maxPerEpoch, rewardAmt)
			maxPayoutPerParticipant = e.global.maxPayoutPerParticipant
		}

		e.log.Info("Rewards: reward pot for epoch with max payout per epoch", logging.Uint64("epoch", epoch.Seq), logging.String("rewardBalance", rewardAmt.String()))

		// no point in doing anything after this point if the reward balance is 0
		if rewardAmt.IsZero() {
			continue
		}

		// calculate the rewards per the reward scheme and reword amount
		po := calculateRewards(num.NewUint(epoch.Seq).String(), account.Asset, accountID, rewardAmt, validatorNormalisedScores, validatorData, e.global.delegatorShare, maxPayoutPerParticipant, e.global.minValStakeUInt, e.rng, e.log)
		if po == nil || po.totalReward.IsZero() {
			continue
		}

		if po.totalReward.IsNegative() {
			e.log.Panic("Rewards: payout overflow")
		}

		if po.totalReward.GT(rewardAmt) {
			e.log.Panic("Rewards: payout total greater than reward amount for epoch", logging.String("payoutTotal", po.totalReward.String()), logging.String("rewardAmountForEpoch", rewardAmt.String()), logging.Uint64("epoch", epoch.Seq))
		}

		payouts = append(payouts, po)
		timeToSend := epoch.EndTime.Add(e.global.payoutDelay)
		e.emitEventsForPayout(ctx, timeToSend, po)
		po.timestamp = timeToSend.UnixNano()

		if e.global.payoutDelay == time.Duration(0) {
			e.distributePayout(ctx, po)
			continue
		}

		_, ok := e.pendingPayouts[timeToSend]
		if !ok {
			e.pendingPayouts[timeToSend] = []*payout{po}
		} else {
			e.pendingPayouts[timeToSend] = append(e.pendingPayouts[timeToSend], po)
		}

		e.rss.changed = true
	}
	return payouts
}

func (e *Engine) calculatePercentageOfTotalReward(amount, totalReward *num.Uint) float64 {
	proportion := amount.ToDecimal().Div(totalReward.ToDecimal())
	pct, _ := proportion.Mul(num.DecimalFromInt64(100)).Float64()
	return pct
}

func (e *Engine) emitEventsForPayout(ctx context.Context, timeToSend time.Time, po *payout) {
	payoutEvents := map[string]*events.RewardPayout{}
	parties := []string{}
	for party, amount := range po.partyToAmount {
		pct := e.calculatePercentageOfTotalReward(amount, po.totalReward)
		payoutEvents[party] = events.NewRewardPayout(ctx, timeToSend.UnixNano(), party, po.epochSeq, po.asset, amount, pct)
		parties = append(parties, party)
	}
	sort.Strings(parties)
	payoutEventSlice := make([]events.Event, 0, len(parties))
	for _, p := range parties {
		payoutEventSlice = append(payoutEventSlice, *payoutEvents[p])
	}
	e.broker.SendBatch(payoutEventSlice)
}

// update the account ids for asset infrastructure fees.
func (e *Engine) updateInfraFeeAccountIDs() {
	e.rewardSchemes[infrstructureFeeSchemeID].RewardPoolAccountIDs = e.collateral.GetInfraFeeAccountIDs()
}

// OnEpochEvent calculates the reward amounts parties get for available reward schemes.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("OnEpochEvent")

	if (epoch.EndTime == time.Time{}) {
		e.epochSeq = num.NewUint(epoch.Seq).String()
		e.newEpochStarted = true
		return
	}

	e.calculateRewardPayouts(ctx, epoch)
}

func (e *Engine) calculateRewardPayouts(ctx context.Context, epoch types.Epoch) []*payout {
	rsIDs := make([]string, 0, len(e.rewardSchemes))
	for rsID := range e.rewardSchemes {
		rsIDs = append(rsIDs, rsID)
	}
	sort.Strings(rsIDs)

	// get the validator delegation data from the delegation engine and calculate the staking and delegation rewards for the epoch
	validatorData := e.delegation.ProcessEpochDelegations(ctx, epoch)

	if e.log.GetLevel() == logging.DebugLevel {
		for _, v := range validatorData {
			e.log.Debug("Rewards: epoch stake summary for validator", logging.Uint64("epoch", epoch.Seq), logging.String("validator", v.NodeID), logging.String("selfStake", v.SelfStake.String()), logging.String("stakeByDelegators", v.StakeByDelegators.String()))
			for party, d := range v.Delegators {
				e.log.Debug("Rewards: epoch delegation for party", logging.Uint64("epoch", epoch.Seq), logging.String("party", party), logging.String("validator", v.NodeID), logging.String("amount", d.String()))
			}
		}
	}

	// calculate the validator score for each validator and the total score for all
	validatorNormalisedScores := calcValidatorsNormalisedScore(ctx, e.broker, num.NewUint(epoch.Seq).String(), validatorData, e.global.minValidators, e.global.compLevel, e.global.optimalStakeMultiplier, e.rng, e.valPerformance)
	for node, score := range validatorNormalisedScores {
		e.log.Info("Rewards: calculated normalised score", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}

	e.updateInfraFeeAccountIDs()
	payouts := []*payout{}

	for _, rsID := range rsIDs {
		rewardScheme := e.rewardSchemes[rsID]

		// if reward scheme is not active yet or anymore, ignore it
		if !rewardScheme.IsActive(epoch.EndTime) {
			continue
		}

		onChainTreasury := rewardScheme.Type == types.RewardSchemeStakingAndDelegation
		payouts = append(payouts, e.processRewards(ctx, rewardScheme, epoch, validatorData, validatorNormalisedScores, onChainTreasury)...)
	}
	return payouts
}

// make the required transfers for distributing reward payout.
func (e *Engine) distributePayout(ctx context.Context, po *payout) {
	partyIDs := make([]string, 0, len(po.partyToAmount))
	for party := range po.partyToAmount {
		partyIDs = append(partyIDs, party)
	}

	sort.Strings(partyIDs)
	transfers := make([]*types.Transfer, 0, len(partyIDs))
	for _, party := range partyIDs {
		amt := po.partyToAmount[party]
		transfers = append(transfers, &types.Transfer{
			Owner: party,
			Amount: &types.FinancialAmount{
				Asset:  po.asset,
				Amount: amt.Clone(),
			},
			Type:      types.TransferTypeRewardPayout,
			MinAmount: amt.Clone(),
		})
	}

	responses, err := e.collateral.TransferRewards(ctx, po.fromAccount, transfers)
	if err != nil {
		e.log.Error("error in transfer rewards", logging.Error(err))
		return
	}
	e.broker.Send(events.NewTransferResponse(ctx, responses))
}

// ValidatorKeyChanged is called when the validator public key (aka party) is changed we need to update all pending information to use the new key.
func (e *Engine) ValidatorKeyChanged(ctx context.Context, oldKey, newKey string) {
	if len(e.pendingPayouts) == 0 {
		return
	}

	payTimes := make([]time.Time, 0, len(e.pendingPayouts))
	for payTime := range e.pendingPayouts {
		payTimes = append(payTimes, payTime)
	}
	payoutEvents := []events.Event{}
	sort.Slice(payTimes, func(i, j int) bool { return payTimes[i].Before(payTimes[j]) })
	for _, payTime := range payTimes {
		// remove all paid payouts from pending
		for _, p := range e.pendingPayouts[payTime] {
			if amount, ok := p.partyToAmount[oldKey]; ok {
				delete(p.partyToAmount, oldKey)
				p.partyToAmount[newKey] = amount
				pct := e.calculatePercentageOfTotalReward(amount, p.totalReward)
				payoutEvents = append(payoutEvents,
					*events.NewRewardPayout(ctx, p.timestamp, oldKey, p.epochSeq, p.asset, num.Zero(), 0),
					*events.NewRewardPayout(ctx, p.timestamp, newKey, p.epochSeq, p.asset, amount, pct),
				)
			}
		}
	}
	e.broker.SendBatch(payoutEvents)
	e.rss.changed = true
}

// shouldUpdateValidatorsVotingPower returns whether we should update the voting power of the validator in tendermint
// currently this should happen at the beginning of each epoch (at the end of the first block of the new epoch) and every 1000 blocks.
func (e *Engine) shouldUpdateValidatorsVotingPower(height int64) bool {
	if e.newEpochStarted {
		e.newEpochStarted = false
		return true
	}
	return height%1000 == 0
}

// EndOfBlock returns the validator updates with the power of the validators based on their stake in the current block.
func (e *Engine) EndOfBlock(blockHeight int64) []types.ValidatorVotingPower {
	if !e.shouldUpdateValidatorsVotingPower(blockHeight) {
		return nil
	}

	validatorsData := e.delegation.GetValidatorData()
	scoreData := calcNormalisedScore(e.epochSeq, validatorsData, e.global.minValidators, e.global.compLevel, e.global.optimalStakeMultiplier, e.rng, e.valPerformance)
	votingPower := make([]types.ValidatorVotingPower, 0, len(validatorsData))
	for _, v := range validatorsData {
		ns, ok := scoreData.normalisedScores[v.NodeID]
		power := int64(10)

		if ok {
			power = num.MaxD(decimal1, ns.Mul(votingPowerScalingFactor)).IntPart()
		}
		votingPower = append(votingPower, types.ValidatorVotingPower{
			VotingPower: power,
			TmPubKey:    v.TmPubKey,
		})
	}

	return votingPower
}
