package validators

import (
	"context"
	"encoding/base64"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/encoding"
)

var (
	ErrUnknownValidator            = errors.New("unknown validator ID")
	ErrUnexpectedSignedBlockHeight = errors.New("unexpected signed block height")

	PerformanceIncrement        = num.DecimalFromFloat(0.1)
	DecimalOne                  = num.DecimalFromFloat(1)
	VotingPowerScalingFactor, _ = num.DecimalFromString("10000")
	BlocksToKeepMalperforming   = int64(1000000)
)

type ValidatorStatus int32

const (
	ValidatorStatusPending = iota
	ValidatorStatusErsatz
	ValidatorStatusTendermint
)

var ValidatorStatusToName = map[ValidatorStatus]string{
	ValidatorStatusPending:    "pending",
	ValidatorStatusErsatz:     "ersatz",
	ValidatorStatusTendermint: "tendermint",
}

type valState struct {
	data                            ValidatorData
	blockAdded                      int64                      // the block it was added to vega
	status                          ValidatorStatus            // the status of the node (tendermint, ersatz, waiting list)
	statusChangeBlock               int64                      // the block id in which it got to its current status
	lastBlockWithPositiveRanking    int64                      // the last epoch with non zero ranking for the validator
	numberOfEthereumEventsForwarded uint64                     // number of events forwarded by the validator
	heartbeatTracker                *validatorHeartbeatTracker // track hearbeat transactions
	validatorPower                  int64
}

// UpdateNumberEthMultisigSigners updates the required number of multisig signers.
func (t *Topology) UpdateNumberEthMultisigSigners(_ context.Context, numberEthMultisigSigners *num.Uint) error {
	t.numberEthMultisigSigners = int(numberEthMultisigSigners.Uint64())
	return nil
}

// UpdateNumberOfTendermintValidators updates with the quota for tendermint validators. It updates accordingly the number of slots for ersatzvalidators.
func (t *Topology) UpdateNumberOfTendermintValidators(_ context.Context, noValidators *num.Uint) error {
	t.numberOfTendermintValidators = int(noValidators.Uint64())
	t.numberOfErsatzValidators = int(t.ersatzValidatorsFactor.Mul(noValidators.ToDecimal()).IntPart())
	return nil
}

// UpdateErsatzValidatorsFactor updates the ratio between the tendermint validators list and the ersatz validators list.
func (t *Topology) UpdateErsatzValidatorsFactor(_ context.Context, ersatzFactor num.Decimal) error {
	t.ersatzValidatorsFactor = ersatzFactor
	t.numberOfErsatzValidators = int(t.ersatzValidatorsFactor.Mul(num.DecimalFromInt64(int64(t.numberOfTendermintValidators))).IntPart())
	return nil
}

// UpdateValidatorIncumbentBonusFactor updates with the net param for incumbent bonus, saved as incumbentBonusFactor + 1.
func (t *Topology) UpdateValidatorIncumbentBonusFactor(_ context.Context, incumbentBonusFactor num.Decimal) error {
	t.validatorIncumbentBonusFactor = DecimalOne.Add(incumbentBonusFactor)
	return nil
}

// UpdateMinimumEthereumEventsForNewValidator updates the minimum number of events forwarded by / voted for by the joining validator.
func (t *Topology) UpdateMinimumEthereumEventsForNewValidator(_ context.Context, minimumEthereumEventsForNewValidator *num.Uint) error {
	t.minimumEthereumEventsForNewValidator = minimumEthereumEventsForNewValidator.Uint64()
	return nil
}

// UpdateMinimumRequireSelfStake updates the minimum requires stake for a validator.
func (t *Topology) UpdateMinimumRequireSelfStake(_ context.Context, minStake num.Decimal) error {
	t.minimumStake, _ = num.UintFromDecimal(minStake)
	return nil
}

// AddForwarder records the times that a validator fowards an eth event.
func (t *Topology) AddForwarder(pubKey string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, vs := range t.validators {
		if vs.data.VegaPubKey == pubKey {
			vs.numberOfEthereumEventsForwarded++
		}
	}
}

// RecalcValidatorSet is called at the before a new epoch is started to update the validator sets.
// the delegation state corresponds to the epoch about to begin.
func (t *Topology) RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) {
	// This can actually change the validators structure so no reads should be allowed in parallel to this.
	t.mu.Lock()
	defer t.mu.Unlock()
	// first we record the current status of validators before the promotion/demotion so we can capture in an event.
	currentState := make(map[string]statusAddress, len(t.validators))
	for k, vs := range t.validators {
		currentState[k] = statusAddress{
			status:     vs.status,
			ethAddress: vs.data.EthereumAddress,
		}
	}

	keys := make([]string, 0, len(currentState))
	for k := range currentState {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// get the ranking of the validators for the purpose of promotion
	stakeScore, perfScore, rankingScore := t.getRankingScore(delegationState)

	// apply promotion logic - returns the tendermint updates with voting power changes (including removals and additions)
	vpu, nextVotingPower := t.applyPromotion(perfScore, rankingScore, delegationState, stakeScoreParams)
	t.validatorPowerUpdates = vpu
	for _, vu := range t.validatorPowerUpdates {
		cPubKey, _ := encoding.PubKeyFromProto(vu.PubKey)
		t.log.Info("setting voting power to", logging.String(("address"), cPubKey.Address().String()), logging.Uint64("power", uint64(vu.Power)))
	}

	newState := make(map[string]statusAddress, len(t.validators))
	for k, vs := range t.validators {
		newState[k] = statusAddress{
			status:     vs.status,
			ethAddress: vs.data.EthereumAddress,
		}
	}

	// do this only if we are a validator, not need otherwise
	if t.IsValidator() {
		t.signatures.EmitPromotionsSignatures(
			ctx, t.currentTime, currentState, newState)
	}

	// prepare and send the events
	evts := make([]events.Event, 0, len(currentState))
	for _, nodeID := range keys {
		status := "removed"
		if vd, ok := t.validators[nodeID]; ok {
			status = ValidatorStatusToName[vd.status]
		}

		vp, ok := nextVotingPower[nodeID]
		if !ok {
			vp = 0
		}

		evts = append(evts, events.NewValidatorRanking(ctx, epochSeq, nodeID, stakeScore[nodeID].String(), perfScore[nodeID].String(), rankingScore[nodeID].String(), ValidatorStatusToName[currentState[nodeID].status], status, int(vp)))
	}
	t.broker.SendBatch(evts)

	nodeIDs := make([]string, 0, len(rankingScore))
	for k := range rankingScore {
		nodeIDs = append(nodeIDs, k)
	}
	sort.Strings(nodeIDs)

	// update the lastBlockWithNonZeroRanking
	for _, k := range nodeIDs {
		d := rankingScore[k]
		if d.IsPositive() {
			t.validators[k].lastBlockWithPositiveRanking = int64(t.currentBlockHeight)
			continue
		}
		// if the node hasn't had a positive score for more than 10 epochs it is dropped - unless it has stake delegated to it, otherwise this stake
		// will be lost
		if int64(t.currentBlockHeight)-t.validators[k].lastBlockWithPositiveRanking > BlocksToKeepMalperforming && stakeScore[k].IsZero() {
			t.log.Info("removing validator with 0 positive ranking for too long", logging.String("node-id", k))
			t.sendValidatorUpdateEvent(ctx, t.validators[k].data, true)
			delete(t.validators, k)
		}
	}
	t.tss.changed = true
}

// applyPromotion calculates the new validator set for tendermint and ersatz and returns the set of updates to apply to tendermint voting powers.
func (t *Topology) applyPromotion(performanceScore, rankingScore map[string]num.Decimal, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) ([]tmtypes.ValidatorUpdate, map[string]int64) {
	tendermintValidators := []*valState{}
	remainingValidators := []*valState{}

	// split the validator set into current tendermint validators and the others
	for _, vd := range t.validators {
		if vd.status == ValidatorStatusTendermint {
			tendermintValidators = append(tendermintValidators, vd)
		} else {
			remainingValidators = append(remainingValidators, vd)
		}
	}

	// sort the tendermint validators in descending order of their ranking score with the earlier block added as a tier breaker
	sort.SliceStable(tendermintValidators, func(i, j int) bool {
		// tiebreaker: the one which was promoted to tm validator first gets higher
		if rankingScore[tendermintValidators[i].data.ID].Equal(rankingScore[tendermintValidators[j].data.ID]) {
			if tendermintValidators[i].statusChangeBlock == tendermintValidators[j].statusChangeBlock {
				return t.rng.Int31n(2) > 0
			}
			return tendermintValidators[i].statusChangeBlock < tendermintValidators[j].statusChangeBlock
		} else {
			return rankingScore[tendermintValidators[i].data.ID].GreaterThan(rankingScore[tendermintValidators[j].data.ID])
		}
	})

	// if there are not enought slots, demote from tm to remaining
	tendermintValidators, remainingValidators, removedFromTM := demoteDueToLackOfSlots(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, t.numberOfTendermintValidators, int64(t.currentBlockHeight+1))

	// now we're sorting the remaining validators - some of which may be eratz, some may have been tendermint (as demoted above) and some just in the waiting list
	sort.SliceStable(remainingValidators, func(i, j int) bool {
		// tiebreaker: the one which submitted their transaction to join earlier
		if rankingScore[remainingValidators[i].data.ID].Equal(rankingScore[remainingValidators[j].data.ID]) {
			if remainingValidators[i].blockAdded == remainingValidators[j].blockAdded {
				return t.rng.Int31n(2) > 0
			}
			return remainingValidators[i].blockAdded < remainingValidators[j].blockAdded
		} else {
			return rankingScore[remainingValidators[i].data.ID].GreaterThan(rankingScore[remainingValidators[j].data.ID])
		}
	})

	// apply promotions and demotions from tendermint to ersatz and vice versa
	tendermintValidators, remainingValidators, demotedFromTM := promote(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, t.numberOfTendermintValidators, rankingScore, int64(t.currentBlockHeight+1))
	removedFromTM = append(removedFromTM, demotedFromTM...)

	// by this point we're done with promotions to tendermint. check if any validator from the waiting list can join the ersatz list
	ersatzValidators := []*valState{}
	waitingListValidators := []*valState{}
	for _, vd := range remainingValidators {
		if vd.status == ValidatorStatusErsatz {
			ersatzValidators = append(ersatzValidators, vd)
		} else if rankingScore[vd.data.ID].IsPositive() {
			waitingListValidators = append(waitingListValidators, vd)
		}
	}

	// demote from ersatz to pending due to more ersatz than slots allowed
	ersatzValidators, waitingListValidators, _ = demoteDueToLackOfSlots(ersatzValidators, waitingListValidators, ValidatorStatusErsatz, ValidatorStatusPending, t.numberOfErsatzValidators, int64(t.currentBlockHeight+1))

	// apply promotions and demotions from ersatz to pending and vice versa
	promote(ersatzValidators, waitingListValidators, ValidatorStatusErsatz, ValidatorStatusPending, t.numberOfErsatzValidators, rankingScore, int64(t.currentBlockHeight+1))

	nextValidators := make([]string, 0, t.numberOfTendermintValidators+len(removedFromTM))
	for _, vd := range t.validators {
		if vd.status == ValidatorStatusTendermint {
			nextValidators = append(nextValidators, vd.data.ID)
		}
	}
	nextValidatorSet := make(map[string]struct{}, len(nextValidators))
	for _, v := range nextValidators {
		nextValidatorSet[v] = struct{}{}
	}

	// extract the delegation and the total delegation of the new set of validators for tendermint
	tmDelegation, tmTotalDelegation := CalcDelegation(nextValidatorSet, delegationState)

	// calculate the anti-whaling stake score of the validators with respect to stake represented by the tm validators
	nextValidatorsStakeScore := CalcAntiWhalingScore(tmDelegation, tmTotalDelegation, stakeScoreParams)

	// recored the performance score of the tm validators
	nextValidatorsPerformanceScore := make(map[string]num.Decimal, len(nextValidatorSet))
	for k := range nextValidatorSet {
		nextValidatorsPerformanceScore[k] = performanceScore[k]
	}
	// calculate the score as stake_score x perf_score (no need to normalise, this will be done inside calculateVotingPower
	nextValidatorsScore := getValScore(nextValidatorsStakeScore, nextValidatorsPerformanceScore)

	// calculate the voting power of the next tendermint validators
	nextValidatorsVotingPower := t.calculateVotingPower(nextValidators, nextValidatorsScore)

	// add the removed validators with 0 voting power
	for _, removed := range removedFromTM {
		nextValidators = append(nextValidators, removed)
		nextValidatorsVotingPower[removed] = 0
	}

	sort.Strings(nextValidators)

	// generate the tendermint updates from the voting power
	vUpdates := make([]tmtypes.ValidatorUpdate, 0, len(nextValidators))

	// make sure we update the validator power to all nodes, so first reset all to 0
	for _, vd := range t.validators {
		vd.validatorPower = 0
	}

	// now update the validator power for the ones that go to tendermint
	for _, v := range nextValidators {
		vd := t.validators[v]
		pubkey, err := base64.StdEncoding.DecodeString(vd.data.TmPubKey)
		if err != nil {
			continue
		}
		vd.validatorPower = nextValidatorsVotingPower[v]
		update := tmtypes.UpdateValidator(pubkey, nextValidatorsVotingPower[v], "")
		vUpdates = append(vUpdates, update)
	}

	for k, d := range rankingScore {
		t.log.Info("ranking score for promotion", logging.String(k, d.String()))
	}

	for _, vu := range vUpdates {
		t.log.Info("voting power update", logging.String("pubKey", vu.PubKey.String()), logging.Int64("power", vu.Power))
	}

	return vUpdates, nextValidatorsVotingPower
}

func demoteDueToLackOfSlots(seriesA []*valState, seriesB []*valState, statusA ValidatorStatus, statusB ValidatorStatus, maxForSeriesA int, nextBlockHeight int64) ([]*valState, []*valState, []string) {
	removedFromSeriesA := []string{}
	// that means that the number of tendermint validators has been reduced and we need to downgrade some validators to become ersatz
	if len(seriesA) > maxForSeriesA {
		demoted := len(seriesA) - maxForSeriesA
		for i := 0; i < demoted; i++ {
			vd := seriesA[len(seriesA)-1-i]
			vd.status = statusB
			vd.statusChangeBlock = nextBlockHeight
			// add to the remaining validators so it can compete with the ersatzvalidators
			seriesB = append(seriesB, vd)
			removedFromSeriesA = append(removedFromSeriesA, vd.data.ID)
		}
		seriesA = seriesA[:maxForSeriesA]
		return seriesA, seriesB, removedFromSeriesA
	}

	return seriesA, seriesB, removedFromSeriesA
}

func promote(seriesA []*valState, seriesB []*valState, statusA ValidatorStatus, statusB ValidatorStatus, maxForSeriesA int, rankingScore map[string]num.Decimal, nextBlockHeight int64) ([]*valState, []*valState, []string) {
	removedFromSeriesA := []string{}

	// there's free slots to become seriesA validator and there are candidates
	if promotion := maxForSeriesA - len(seriesA); promotion > 0 && len(seriesB) > 0 {
		if promotion > len(seriesB) {
			promotion = len(seriesB)
		}
		for i := 0; i < promotion; i++ {
			if rankingScore[seriesB[i].data.ID].IsPositive() {
				seriesB[i].statusChangeBlock = nextBlockHeight
				seriesB[i].status = statusA
			} else {
				break
			}
		}
	} else if maxForSeriesA > 0 && maxForSeriesA == len(seriesA) && len(seriesB) > 0 {
		// the best of the remaining is better than the worst tendermint validator
		if rankingScore[seriesA[len(seriesA)-1].data.ID].LessThan(rankingScore[seriesB[0].data.ID]) {
			vd := seriesA[len(seriesA)-1]
			vd.status = statusB
			vd.statusChangeBlock = nextBlockHeight
			removedFromSeriesA = append(removedFromSeriesA, vd.data.ID)

			vd = seriesB[0]
			vd.status = statusA
			vd.statusChangeBlock = nextBlockHeight
		}
	}
	return seriesA, seriesB, removedFromSeriesA
}

// calculateVotingPower returns the voting powers as the normalised ranking scores scaled by VotingPowerScalingFactor with a minimum of 1.
func (t *Topology) calculateVotingPower(IDs []string, rankingScores map[string]num.Decimal) map[string]int64 {
	votingPower := make(map[string]int64, len(IDs))
	sumOfScores := num.DecimalZero()
	for _, ID := range IDs {
		sumOfScores = sumOfScores.Add(rankingScores[ID])
	}

	for _, ID := range IDs {
		if sumOfScores.IsPositive() && rankingScores[ID].IsPositive() {
			votingPower[ID] = num.MaxD(DecimalOne, rankingScores[ID].Div(sumOfScores).Mul(VotingPowerScalingFactor)).IntPart()
		} else {
			votingPower[ID] = 10
		}
	}
	return votingPower
}

// GetValidatorPowerUpdates returns the voting power changes if this is the first block of an epoch.
func (t *Topology) GetValidatorPowerUpdates() []tmtypes.ValidatorUpdate {
	if t.newEpochStarted {
		t.newEpochStarted = false
		return t.validatorPowerUpdates
	}
	return []tmtypes.ValidatorUpdate{}
}
