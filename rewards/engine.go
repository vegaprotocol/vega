package rewards

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

var (
	decimal1, _        = num.DecimalFromString("1")
	rewardAccountTypes = []types.AccountType{types.AccountTypeGlobalReward, types.AccountTypeFeesInfrastructure, types.AccountTypeMakerFeeReward, types.AccountTypeTakerFeeReward, types.AccountTypeLPFeeReward}
)

// Broker for sending events.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/fees_tracker_mock.go -package mocks code.vegaprotocol.io/vega/rewards FeesTracker
type FeesTracker interface {
	GetFeePartyScores(asset string, feeType types.TransferType) []*types.FeePartyScore
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_tracker_mock.go -package mocks code.vegaprotocol.io/vega/rewards MarketTracker
type MarketTracker interface {
	GetAndResetEligibleProposers() []string
}

// EpochEngine notifies the reward engine at the end of an epoch.
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch))
	NotifyOnEpochRestore(f func(context.Context, types.Epoch))
}

//Delegation engine for getting validation data
//go:generate go run github.com/golang/mock/mockgen -destination mocks/delegation_engine_mock.go -package mocks code.vegaprotocol.io/vega/rewards Delegation
type Delegation interface {
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	GetValidatorData() []*types.ValidatorData
}

// Collateral engine provides access to account data and transferring rewards.
type Collateral interface {
	GetAccountByID(id string) (*types.Account, error)
	TransferRewards(ctx context.Context, rewardAccountID string, transfers []*types.Transfer) ([]*types.TransferResponse, error)
	GetInfraFeeAccountIDs() []string
	GetEnabledAssets() []string
	GetRewardAccount(asset string, rewardAcccountType types.AccountType) (*types.Account, error)
}

//TimeService notifies the reward engine on time updates
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/rewards TimeService
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
	GetTimeNow() time.Time
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/topology_mock.go -package mocks code.vegaprotocol.io/vega/rewards Topology
type Topology interface {
	GetRewardsScores(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData)
	RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams)
}

// Engine is the reward engine handling reward payouts.
type Engine struct {
	log                *logging.Logger
	config             Config
	broker             Broker
	topology           Topology
	delegation         Delegation
	collateral         Collateral
	feesTracker        FeesTracker
	marketTracker      MarketTracker
	rng                *rand.Rand
	global             *globalRewardParams
	newEpochStarted    bool // flag to signal new epoch so we can update the voting power at the end of the block
	epochSeq           string
	ersatzRewardFactor num.Decimal
}

type globalRewardParams struct {
	minValStakeD            num.Decimal
	minValStakeUInt         *num.Uint
	optimalStakeMultiplier  num.Decimal
	compLevel               num.Decimal
	minValidators           num.Decimal
	maxPayoutPerParticipant *num.Uint
	delegatorShare          num.Decimal
	asset                   string
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
func New(log *logging.Logger, config Config, broker Broker, delegation Delegation, epochEngine EpochEngine, collateral Collateral, ts TimeService, feesTracker FeesTracker, marketTracker MarketTracker, topology Topology) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:          config,
		log:             log.Named(namedLogger),
		broker:          broker,
		delegation:      delegation,
		collateral:      collateral,
		global:          &globalRewardParams{},
		newEpochStarted: false,
		feesTracker:     feesTracker,
		marketTracker:   marketTracker,
		topology:        topology,
	}

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent)
	epochEngine.NotifyOnEpochRestore(e.OnEpochRestore)

	// register for time tick updates
	ts.NotifyOnTick(e.onChainTimeUpdate)
	return e
}

func (e *Engine) UpdateAssetForStakingAndDelegation(ctx context.Context, asset string) error {
	e.global.asset = asset
	return nil
}

// UpdateErsatzRewardFactor updates the ratio of staking and delegation reward that goes to ersatz validators.
func (e *Engine) UpdateErsatzRewardFactor(ctx context.Context, ersatzRewardFactor num.Decimal) error {
	e.ersatzRewardFactor = ersatzRewardFactor
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

// UpdateMaxPayoutPerParticipantForStakingRewardScheme is a callback for changes in the network param for max payout per participant.
func (e *Engine) UpdateMaxPayoutPerParticipantForStakingRewardScheme(ctx context.Context, maxPayoutPerParticipant num.Decimal) error {
	e.global.maxPayoutPerParticipant, _ = num.UintFromDecimal(maxPayoutPerParticipant)
	return nil
}

// UpdateDelegatorShareForStakingRewardScheme is a callback for changes in the network param for delegator share.
func (e *Engine) UpdateDelegatorShareForStakingRewardScheme(ctx context.Context, delegatorShare num.Decimal) error {
	e.global.delegatorShare = delegatorShare
	return nil
}

// whenever we have a time update, check if there are pending payouts ready to be sent.
func (e *Engine) onChainTimeUpdate(ctx context.Context, t time.Time) {
	// resetting the seed every block, to both get some more unpredictability and still deterministic
	// and play nicely with snapshot
	e.rng = rand.New(rand.NewSource(t.Unix()))
}

// OnEpochEvent calculates the reward amounts parties get for available reward schemes.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("OnEpochEvent")

	// on new epoch update the epoch seq and update the epoch started flag
	if (epoch.EndTime == time.Time{}) {
		e.epochSeq = num.NewUint(epoch.Seq).String()
		e.newEpochStarted = true
		return
	}

	// we're at the end of the epoch - process rewards
	e.calculateRewardPayouts(ctx, epoch)
}

func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("epoch restoration notification received", logging.String("epoch", epoch.String()))
	e.epochSeq = num.NewUint(epoch.Seq).String()
	e.newEpochStarted = true
}

// splitDelegationByStatus splits the delegation data for an epoch into tendermint and ersatz validator sets.
func (e *Engine) splitDelegationByStatus(delegation []*types.ValidatorData, tmScores *types.ScoreData, ezScores *types.ScoreData) ([]*types.ValidatorData, []*types.ValidatorData) {
	tm := make([]*types.ValidatorData, 0, len(tmScores.NodeIDSlice))
	ez := make([]*types.ValidatorData, 0, len(ezScores.NodeIDSlice))
	for _, vd := range delegation {
		if _, ok := tmScores.NormalisedScores[vd.NodeID]; ok {
			tm = append(tm, vd)
		}
		if _, ok := ezScores.NormalisedScores[vd.NodeID]; ok {
			ez = append(ez, vd)
		}
	}
	return tm, ez
}

func calcTotalDelegation(d []*types.ValidatorData) num.Decimal {
	total := num.Zero()
	for _, vd := range d {
		total.AddSum(num.Sum(vd.SelfStake, vd.StakeByDelegators))
	}
	return total.ToDecimal()
}

func (e *Engine) calculateRewardPayouts(ctx context.Context, epoch types.Epoch) []*payout {
	// get the validator delegation data from the delegation engine and calculate the staking and delegation rewards for the epoch
	delegationState := e.delegation.ProcessEpochDelegations(ctx, epoch)

	stakeScoreParams := types.StakeScoreParams{MinVal: e.global.minValidators, CompLevel: e.global.compLevel, OptimalStakeMultiplier: e.global.optimalStakeMultiplier}

	// NB: performance scores for rewards are calculated with the current values of the voting power
	tmValidatorsScores, ersatzValidatorsScores := e.topology.GetRewardsScores(ctx, e.epochSeq, delegationState, stakeScoreParams)
	tmValidatorsDelegation, ersatzValidatorsDelegation := e.splitDelegationByStatus(delegationState, tmValidatorsScores, ersatzValidatorsScores)

	// let the topology process the changes in delegation set and calculate changes to tendermint/ersatz validator sets
	// again, performance scores for ranking is based on the current voting powers.
	// performance data will be erased in the next block which is the first block of the new epoch
	e.topology.RecalcValidatorSet(ctx, num.NewUint(epoch.Seq+1).String(), e.delegation.GetValidatorData(), stakeScoreParams)

	s_p := calcTotalDelegation(tmValidatorsDelegation)
	s_e := calcTotalDelegation(ersatzValidatorsDelegation).Mul(e.ersatzRewardFactor)
	s_t := s_p.Add(s_e)
	s_pFactor := num.DecimalZero()
	s_eFactor := num.DecimalZero()
	// if there's stake calculate the factors of primary vs ersatz and make sure it's <= 1
	if s_t.IsPositive() {
		s_pFactor = s_p.Div(s_t)
		s_eFactor = s_e.Div(s_t)
		overflow := num.MinD(num.DecimalZero(), s_pFactor.Add(s_eFactor).Sub(decimal1))
		s_eFactor = s_eFactor.Sub(overflow)
	}

	for node, score := range tmValidatorsScores.NormalisedScores {
		e.log.Info("Rewards: calculated normalised score for tendermint validators", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}
	for node, score := range ersatzValidatorsScores.NormalisedScores {
		e.log.Info("Rewards: calculated normalised score for ersatz validator", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}

	payouts := []*payout{}
	// all reward types are implicitly defined for all assets. if the balance of the reward account is non-zero a reward is paid
	for _, asset := range e.collateral.GetEnabledAssets() {
		for _, rewardType := range rewardAccountTypes {
			account, err := e.collateral.GetRewardAccount(asset, rewardType)

			if err != nil || account.Balance.IsZero() {
				continue
			}
			pos := []*payout{}
			if (rewardType == types.AccountTypeGlobalReward && asset == e.global.asset) || rewardType == types.AccountTypeFeesInfrastructure {
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), asset, rewardType, account, tmValidatorsDelegation, tmValidatorsScores.NormalisedScores, epoch.EndTime, s_pFactor))
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), asset, rewardType, account, ersatzValidatorsDelegation, ersatzValidatorsScores.NormalisedScores, epoch.EndTime, s_eFactor))
			} else {
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), asset, rewardType, account, tmValidatorsDelegation, tmValidatorsScores.NormalisedScores, epoch.EndTime, decimal1))
			}
			for _, po := range pos {
				if po != nil && !po.totalReward.IsZero() && !po.totalReward.IsNegative() {
					po.timestamp = epoch.EndTime.UnixNano()
					payouts = append(payouts, po)
					e.emitEventsForPayout(ctx, epoch.EndTime, po)
					e.distributePayout(ctx, po)
				}
			}
		}
	}
	return payouts
}

// calculateRewardTypeForAsset calculates the payout for a given asset and reward type.
func (e *Engine) calculateRewardTypeForAsset(epochSeq string, asset string, rewardType types.AccountType, account *types.Account, validatorData []*types.ValidatorData, validatorNormalisedScores map[string]num.Decimal, timestamp time.Time, factor num.Decimal) *payout {
	switch rewardType {
	case types.AccountTypeGlobalReward: // given to delegator based on stake
		if asset == e.global.asset {
			balance, _ := num.UintFromDecimal(account.Balance.ToDecimal().Mul(factor))
			return calculateRewardsByStake(epochSeq, account.Asset, account.ID, balance, validatorNormalisedScores, validatorData, e.global.delegatorShare, e.global.maxPayoutPerParticipant, e.global.minValStakeUInt, e.rng, e.log)
		}
		return nil
	case types.AccountTypeFeesInfrastructure: // given to delegator based on stake
		return calculateRewardsByStake(epochSeq, account.Asset, account.ID, account.Balance.Clone(), validatorNormalisedScores, validatorData, e.global.delegatorShare, num.Zero(), e.global.minValStakeUInt, e.rng, e.log)
	case types.AccountTypeMakerFeeReward: // given to receivers of maker fee in the asset based on their total received fee proportion
		return calculateRewardsByContribution(epochSeq, account.Asset, account.ID, rewardType, account.Balance, e.feesTracker.GetFeePartyScores(asset, types.TransferTypeMakerFeeReceive), timestamp)
	case types.AccountTypeTakerFeeReward: // given to payers of fee in the asset based on their total paid fee proportion
		return calculateRewardsByContribution(epochSeq, account.Asset, account.ID, rewardType, account.Balance, e.feesTracker.GetFeePartyScores(asset, types.TransferTypeMakerFeePay), timestamp)
	case types.AccountTypeLPFeeReward: // given to LP fee receivers in the asset based on their total received fee
		return calculateRewardsByContribution(epochSeq, account.Asset, account.ID, rewardType, account.Balance, e.feesTracker.GetFeePartyScores(asset, types.TransferTypeLiquidityFeeDistribute), timestamp)
	case types.AccountTypeMarketProposerReward:
		return calculateRewardForProposers(epochSeq, account.Asset, account.ID, rewardType, account.Balance, e.marketTracker.GetAndResetEligibleProposers(), timestamp)
	}
	return nil
}

// emitEventsForPayout fires events corresponding to the reward payout.
func (e *Engine) emitEventsForPayout(ctx context.Context, timeToSend time.Time, po *payout) {
	payoutEvents := map[string]*events.RewardPayout{}
	parties := []string{}
	totalReward := po.totalReward.ToDecimal()
	for party, amount := range po.partyToAmount {
		proportion := amount.ToDecimal().Div(totalReward)
		pct := proportion.Mul(num.DecimalFromInt64(100))
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

// distributePayout creates a set of transfers corresponding to a reward payout.
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
