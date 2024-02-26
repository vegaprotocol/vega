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

package rewards

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

var (
	decimal1, _        = num.DecimalFromString("1")
	rewardAccountTypes = []types.AccountType{types.AccountTypeGlobalReward, types.AccountTypeFeesInfrastructure, types.AccountTypeMakerReceivedFeeReward, types.AccountTypeMakerPaidFeeReward, types.AccountTypeLPFeeReward, types.AccountTypeMarketProposerReward, types.AccountTypeAveragePositionReward, types.AccountTypeRelativeReturnReward, types.AccountTypeReturnVolatilityReward, types.AccountTypeValidatorRankingReward}
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/rewards MarketActivityTracker,Delegation,TimeService,Topology,Transfers,Teams,Vesting,ActivityStreak

// Broker for sending events.
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

type MarketActivityTracker interface {
	GetAllMarketIDs() []string
	GetProposer(market string) string
	CalculateMetricForIndividuals(ds *vega.DispatchStrategy) []*types.PartyContributionScore
	CalculateMetricForTeams(ds *vega.DispatchStrategy) ([]*types.PartyContributionScore, map[string][]*types.PartyContributionScore)
	GetLastEpochTakeFees(asset string, market []string) map[string]*num.Uint
}

// EpochEngine notifies the reward engine at the end of an epoch.
type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

// Delegation engine for getting validation data.
type Delegation interface {
	ProcessEpochDelegations(ctx context.Context, epoch types.Epoch) []*types.ValidatorData
	GetValidatorData() []*types.ValidatorData
}

// Collateral engine provides access to account data and transferring rewards.
type Collateral interface {
	GetAccountByID(id string) (*types.Account, error)
	TransferRewards(ctx context.Context, rewardAccountID string, transfers []*types.Transfer, rewardType types.AccountType) ([]*types.LedgerMovement, error)
	GetRewardAccountsByType(rewardAcccountType types.AccountType) []*types.Account
	GetAssetQuantum(asset string) (num.Decimal, error)
}

// TimeService notifies the reward engine on time updates.
type TimeService interface {
	GetTimeNow() time.Time
}

type Topology interface {
	GetRewardsScores(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) (*types.ScoreData, *types.ScoreData)
	RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) []*types.PartyContributionScore
}

type Transfers interface {
	GetDispatchStrategy(string) *proto.DispatchStrategy
}

type Teams interface {
	GetTeamMembers([]string) map[string][]string
	GetAllPartiesInTeams() []string
}

type Vesting interface {
	AddReward(party, asset string, amount *num.Uint, lockedForEpochs uint64)
	GetRewardBonusMultiplier(party string) (*num.Uint, num.Decimal)
}

type ActivityStreak interface {
	GetRewardsDistributionMultiplier(party string) num.Decimal
}

// Engine is the reward engine handling reward payouts.
type Engine struct {
	log                   *logging.Logger
	config                Config
	timeService           TimeService
	broker                Broker
	topology              Topology
	delegation            Delegation
	collateral            Collateral
	marketActivityTracker MarketActivityTracker
	global                *globalRewardParams
	newEpochStarted       bool // flag to signal new epoch so we can update the voting power at the end of the block
	epochSeq              string
	ersatzRewardFactor    num.Decimal
	vesting               Vesting
	transfers             Transfers
	activityStreak        ActivityStreak
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
	rewardType       types.AccountType
	fromAccount      string
	asset            string
	partyToAmount    map[string]*num.Uint
	totalReward      *num.Uint
	epochSeq         string
	timestamp        int64
	gameID           *string
	lockedForEpochs  uint64
	lockedUntilEpoch string
}

// New instantiate a new rewards engine.
func New(log *logging.Logger, config Config, broker Broker, delegation Delegation, epochEngine EpochEngine, collateral Collateral, ts TimeService, marketActivityTracker MarketActivityTracker, topology Topology, vesting Vesting, transfers Transfers, activityStreak ActivityStreak) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	e := &Engine{
		config:                config,
		log:                   log.Named(namedLogger),
		timeService:           ts,
		broker:                broker,
		delegation:            delegation,
		collateral:            collateral,
		global:                &globalRewardParams{},
		newEpochStarted:       false,
		marketActivityTracker: marketActivityTracker,
		topology:              topology,
		vesting:               vesting,
		transfers:             transfers,
		activityStreak:        activityStreak,
	}

	// register for epoch end notifications
	epochEngine.NotifyOnEpoch(e.OnEpochEvent, e.OnEpochRestore)

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

// OnEpochEvent calculates the reward amounts parties get for available reward schemes.
func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	e.log.Debug("OnEpochEvent")

	// on new epoch update the epoch seq and update the epoch started flag
	if epoch.Action == proto.EpochAction_EPOCH_ACTION_START {
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
	total := num.UintZero()
	for _, vd := range d {
		total.AddSum(num.Sum(vd.SelfStake, vd.StakeByDelegators))
	}
	return total.ToDecimal()
}

// calculateRewardFactors calculates the fraction of the reward given to tendermint and ersatz validators based on their scaled stake.
func (e *Engine) calculateRewardFactors(sp, se num.Decimal) (num.Decimal, num.Decimal) {
	st := sp.Add(se)
	spFactor := num.DecimalZero()
	seFactor := num.DecimalZero()
	// if there's stake calculate the factors of primary vs ersatz and make sure it's <= 1
	if st.IsPositive() {
		spFactor = sp.Div(st)
		seFactor = se.Div(st)
		// if the factors add to more than 1, subtract the excess from the ersatz factors to make the total 1
		overflow := num.MaxD(num.DecimalZero(), spFactor.Add(seFactor).Sub(decimal1))
		seFactor = seFactor.Sub(overflow)
	}

	e.log.Info("tendermint/ersatz fractions of the reward", logging.String("total-delegation", st.String()), logging.String("tenderming-total-delegation", sp.String()), logging.String("ersatz-total-delegation", se.String()), logging.String("tenderming-factor", spFactor.String()), logging.String("ersatz-factor", seFactor.String()))
	return spFactor, seFactor
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
	rankingScoresContributions := e.topology.RecalcValidatorSet(ctx, num.NewUint(epoch.Seq+1).String(), e.delegation.GetValidatorData(), stakeScoreParams)

	sp := calcTotalDelegation(tmValidatorsDelegation)
	se := calcTotalDelegation(ersatzValidatorsDelegation).Mul(e.ersatzRewardFactor)
	spFactor, seFactor := e.calculateRewardFactors(sp, se)
	for node, score := range tmValidatorsScores.NormalisedScores {
		e.log.Info("Rewards: calculated normalised score for tendermint validators", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}
	for node, score := range ersatzValidatorsScores.NormalisedScores {
		e.log.Info("Rewards: calculated normalised score for ersatz validator", logging.String("validator", node), logging.String("normalisedScore", score.String()))
	}

	now := e.timeService.GetTimeNow()
	payouts := []*payout{}
	for _, rewardType := range rewardAccountTypes {
		accounts := e.collateral.GetRewardAccountsByType(rewardType)
		for _, account := range accounts {
			if account.Balance.IsZero() {
				continue
			}
			pos := []*payout{}
			if (rewardType == types.AccountTypeGlobalReward && account.Asset == e.global.asset) || rewardType == types.AccountTypeFeesInfrastructure {
				e.log.Info("calculating reward for tendermint validators", logging.String("account-type", rewardType.String()))
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), account.Asset, rewardType, account, tmValidatorsDelegation, tmValidatorsScores.NormalisedScores, epoch.EndTime, spFactor, rankingScoresContributions))
				e.log.Info("calculating reward for ersatz validators", logging.String("account-type", rewardType.String()))
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), account.Asset, rewardType, account, ersatzValidatorsDelegation, ersatzValidatorsScores.NormalisedScores, epoch.EndTime, seFactor, rankingScoresContributions))
			} else {
				pos = append(pos, e.calculateRewardTypeForAsset(num.NewUint(epoch.Seq).String(), account.Asset, rewardType, account, tmValidatorsDelegation, tmValidatorsScores.NormalisedScores, epoch.EndTime, decimal1, rankingScoresContributions))
			}
			for _, po := range pos {
				if po != nil && !po.totalReward.IsZero() && !po.totalReward.IsNegative() {
					po.rewardType = rewardType
					if account.MarketID != "!" {
						po.gameID = &account.MarketID
					}
					po.timestamp = now.UnixNano()
					payouts = append(payouts, po)
					e.distributePayout(ctx, po)
					po.lockedUntilEpoch = num.NewUint(po.lockedForEpochs + epoch.Seq).String()
					e.emitEventsForPayout(ctx, now, po)
				}
			}
		}
	}

	return payouts
}

func (e *Engine) convertTakerFeesToRewardAsset(takerFees map[string]*num.Uint, fromAsset string, toAsset string) map[string]*num.Uint {
	out := make(map[string]*num.Uint, len(takerFees))
	fromQuantum, err := e.collateral.GetAssetQuantum(fromAsset)
	if err != nil {
		return out
	}
	toQuantum, err := e.collateral.GetAssetQuantum(toAsset)
	if err != nil {
		return out
	}

	quantumRatio := toQuantum.Div(fromQuantum)
	for k, u := range takerFees {
		toAssetAmt, _ := num.UintFromDecimal(u.ToDecimal().Mul(quantumRatio))
		out[k] = toAssetAmt
	}
	return out
}

func (e *Engine) getRewardMultiplierForParty(party string) num.Decimal {
	asMultiplier := e.activityStreak.GetRewardsDistributionMultiplier(party)
	_, vsMultiplier := e.vesting.GetRewardBonusMultiplier(party)
	return asMultiplier.Mul(vsMultiplier)
}

// calculateRewardTypeForAsset calculates the payout for a given asset and reward type.
// for market based rewards, we only care about account for specific markets (as opposed to global account for an asset).
func (e *Engine) calculateRewardTypeForAsset(epochSeq, asset string, rewardType types.AccountType, account *types.Account, validatorData []*types.ValidatorData, validatorNormalisedScores map[string]num.Decimal, timestamp time.Time, factor num.Decimal, rankingScoresContributions []*types.PartyContributionScore) *payout {
	switch rewardType {
	case types.AccountTypeGlobalReward: // given to delegator based on stake
		if asset == e.global.asset {
			balance, _ := num.UintFromDecimal(account.Balance.ToDecimal().Mul(factor))
			e.log.Info("reward balance", logging.String("epoch", epochSeq), logging.String("reward-type", rewardType.String()), logging.String("account-balance", account.Balance.String()), logging.String("factor", factor.String()), logging.String("effective-balance", balance.String()))
			return calculateRewardsByStake(epochSeq, account.Asset, account.ID, balance, validatorNormalisedScores, validatorData, e.global.delegatorShare, e.global.maxPayoutPerParticipant, e.log)
		}
		return nil
	case types.AccountTypeFeesInfrastructure: // given to delegator based on stake
		balance, _ := num.UintFromDecimal(account.Balance.ToDecimal().Mul(factor))
		e.log.Info("reward balance", logging.String("epoch", epochSeq), logging.String("reward-type", rewardType.String()), logging.String("account-balance", account.Balance.String()), logging.String("factor", factor.String()), logging.String("effective-balance", balance.String()))
		return calculateRewardsByStake(epochSeq, account.Asset, account.ID, balance, validatorNormalisedScores, validatorData, e.global.delegatorShare, num.UintZero(), e.log)
	case types.AccountTypeMakerReceivedFeeReward, types.AccountTypeMakerPaidFeeReward, types.AccountTypeLPFeeReward, types.AccountTypeAveragePositionReward, types.AccountTypeRelativeReturnReward, types.AccountTypeReturnVolatilityReward:
		ds := e.transfers.GetDispatchStrategy(account.MarketID)
		if ds == nil {
			return nil
		}
		var takerFeesPaidInRewardAsset map[string]*num.Uint
		if ds.CapRewardFeeMultiple != nil {
			takerFeesPaid := e.marketActivityTracker.GetLastEpochTakeFees(ds.AssetForMetric, ds.Markets)
			takerFeesPaidInRewardAsset = e.convertTakerFeesToRewardAsset(takerFeesPaid, ds.AssetForMetric, asset)
		}
		if ds.EntityScope == vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS {
			partyScores := e.marketActivityTracker.CalculateMetricForIndividuals(ds)
			partyRewardFactors := map[string]num.Decimal{}
			for _, pcs := range partyScores {
				partyRewardFactors[pcs.Party] = e.getRewardMultiplierForParty(pcs.Party)
			}
			return calculateRewardsByContributionIndividual(epochSeq, account.Asset, account.ID, account.Balance, partyScores, partyRewardFactors, timestamp, ds, takerFeesPaidInRewardAsset)
		} else {
			teamScores, partyScores := e.marketActivityTracker.CalculateMetricForTeams(ds)
			partyRewardFactors := map[string]num.Decimal{}
			for _, team := range partyScores {
				for _, pcs := range team {
					partyRewardFactors[pcs.Party] = e.getRewardMultiplierForParty(pcs.Party)
				}
			}
			return calculateRewardsByContributionTeam(epochSeq, account.Asset, account.ID, account.Balance, teamScores, partyScores, partyRewardFactors, timestamp, ds, takerFeesPaidInRewardAsset)
		}

	case types.AccountTypeMarketProposerReward:
		p := calculateRewardForProposers(epochSeq, account.Asset, account.ID, account.Balance, e.marketActivityTracker.GetProposer(account.MarketID), timestamp)
		return p
	case types.AccountTypeValidatorRankingReward:
		ds := e.transfers.GetDispatchStrategy(account.MarketID)
		if ds == nil {
			return nil
		}
		return calculateRewardsForValidators(epochSeq, account.Asset, account.ID, account.Balance, timestamp, rankingScoresContributions, ds.LockPeriod)
	}

	return nil
}

func (e *Engine) emitEventsForPayout(ctx context.Context, timeToSend time.Time, po *payout) {
	payoutEvents := map[string]*events.RewardPayout{}
	parties := []string{}
	totalReward := po.totalReward.ToDecimal()
	assetQuantum, _ := e.collateral.GetAssetQuantum(po.asset)
	for party, amount := range po.partyToAmount {
		proportion := amount.ToDecimal().Div(totalReward)
		pct := proportion.Mul(num.DecimalFromInt64(100))
		payoutEvents[party] = events.NewRewardPayout(ctx, timeToSend.UnixNano(), party, po.epochSeq, po.asset, amount, assetQuantum, pct, po.rewardType, po.gameID, po.lockedUntilEpoch)
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

	responses, err := e.collateral.TransferRewards(ctx, po.fromAccount, transfers, po.rewardType)
	if err != nil {
		e.log.Error("error in transfer rewards", logging.Error(err))
		return
	}

	// if the reward type is not infra fee, report it to the vesting engine
	if po.rewardType != types.AccountTypeFeesInfrastructure {
		for _, party := range partyIDs {
			amt := po.partyToAmount[party]
			e.vesting.AddReward(party, po.asset, amt, po.lockedForEpochs)
		}
	}
	e.broker.Send(events.NewLedgerMovements(ctx, responses))
}
