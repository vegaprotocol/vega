// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	proto "code.vegaprotocol.io/vega/protos/vega"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/encoding"
)

var (
	ErrUnknownValidator            = errors.New("unknown validator ID")
	ErrUnexpectedSignedBlockHeight = errors.New("unexpected signed block height")

	PerformanceIncrement        = num.DecimalFromFloat(0.1)
	DecimalOne                  = num.DecimalFromFloat(1)
	VotingPowerScalingFactor, _ = num.DecimalFromString("10000")
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
	validatorPower                  int64                      // the voting power of the validator
	rankingScore                    *proto.RankingScore        // the last ranking score of the validator
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
func (t *Topology) RecalcValidatorSet(ctx context.Context, epochSeq string, delegationState []*types.ValidatorData, stakeScoreParams types.StakeScoreParams) []*types.PartyContibutionScore {
	// This can actually change the validators structure so no reads should be allowed in parallel to this.
	t.mu.Lock()
	defer t.mu.Unlock()

	consensusValidatorsRankingScores := []*types.PartyContibutionScore{}

	// first we record the current status of validators before the promotion/demotion so we can capture in an event.
	currentState := make(map[string]StatusAddress, len(t.validators))
	for k, vs := range t.validators {
		currentState[k] = StatusAddress{
			Status:           vs.status,
			EthAddress:       vs.data.EthereumAddress,
			SubmitterAddress: vs.data.SubmitterAddress,
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

	newState := make(map[string]StatusAddress, len(t.validators))
	for k, vs := range t.validators {
		newState[k] = StatusAddress{
			Status:           vs.status,
			EthAddress:       vs.data.EthereumAddress,
			SubmitterAddress: vs.data.SubmitterAddress,
		}
	}

	t.signatures.PreparePromotionsSignatures(ctx, t.timeService.GetTimeNow(), t.epochSeq, currentState, newState)

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

		if vd, ok := t.validators[nodeID]; ok {
			vd.rankingScore = &proto.RankingScore{
				StakeScore:       stakeScore[nodeID].String(),
				PerformanceScore: perfScore[nodeID].String(),
				RankingScore:     rankingScore[nodeID].String(),
				PreviousStatus:   statusToProtoStatus(ValidatorStatusToName[currentState[nodeID].Status]),
				Status:           statusToProtoStatus(status),
				VotingPower:      uint32(vp),
			}
			if vd.rankingScore.Status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT {
				consensusValidatorsRankingScores = append(consensusValidatorsRankingScores, &types.PartyContibutionScore{Party: vd.data.VegaPubKey, Score: rankingScore[nodeID]})
			}
		}

		evts = append(evts, events.NewValidatorRanking(ctx, epochSeq, nodeID, stakeScore[nodeID].String(), perfScore[nodeID].String(), rankingScore[nodeID].String(), ValidatorStatusToName[currentState[nodeID].Status], status, int(vp)))
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

		if t.validators[k].status == ValidatorStatusTendermint {
			continue // can't kick out tendermint validator
		}

		if t.validators[k].status == ValidatorStatusPending && (t.validators[k].data.FromEpoch+10) > t.epochSeq {
			continue // pending validators have 10 epochs from when they started their heartbeats to get a positive perf score
		}

		if !stakeScore[k].IsZero() {
			continue // it has stake, we can't kick it out it'll get lost
		}

		// if the node hasn't had a positive score for more than 10 epochs it is dropped
		if int64(t.currentBlockHeight)-t.validators[k].lastBlockWithPositiveRanking > t.blocksToKeepMalperforming {
			t.log.Info("removing validator with 0 positive ranking for too long", logging.String("node-id", k))
			t.validators[k].data.FromEpoch = t.epochSeq
			t.sendValidatorUpdateEvent(ctx, t.validators[k].data, false)
			delete(t.validators, k)
		}
	}
	return consensusValidatorsRankingScores
}

func protoStatusToString(status proto.ValidatorNodeStatus) string {
	if status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT {
		return "tendermint"
	}
	if status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ {
		return "ersatz"
	}
	return "pending"
}

func statusToProtoStatus(status string) proto.ValidatorNodeStatus {
	if status == "tendermint" {
		return proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT
	}
	if status == "ersatz" {
		return proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ
	}
	return proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_PENDING
}

func sortValidatorDescRankingScoreAscBlockcompare(validators []*valState, rankingScore map[string]num.Decimal, blockComparator func(*valState, *valState) bool, rng *rand.Rand) {
	// because we may need the bit of randomness in the sorting below - we need to start from all of the validators in a consistent order
	sort.SliceStable(validators, func(i, j int) bool { return validators[i].data.ID < validators[j].data.ID })

	// sort the tendermint validators in descending order of their ranking score with the earlier block added as a tier breaker
	sort.SliceStable(validators, func(i, j int) bool {
		// tiebreaker: the one which was promoted to tm validator first gets higher
		if rankingScore[validators[i].data.ID].Equal(rankingScore[validators[j].data.ID]) {
			if validators[i].statusChangeBlock == validators[j].statusChangeBlock {
				return rng.Int31n(2) > 0
			}
			return blockComparator(validators[i], validators[j])
		}
		return rankingScore[validators[i].data.ID].GreaterThan(rankingScore[validators[j].data.ID])
	})
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
	byStatusChangeBlock := func(val1, val2 *valState) bool { return val1.statusChangeBlock < val2.statusChangeBlock }
	byBlockAdded := func(val1, val2 *valState) bool { return val1.blockAdded < val2.blockAdded }
	sortValidatorDescRankingScoreAscBlockcompare(tendermintValidators, rankingScore, byStatusChangeBlock, t.rng)
	sortValidatorDescRankingScoreAscBlockcompare(remainingValidators, rankingScore, byBlockAdded, t.rng)

	signers := map[string]struct{}{}
	for _, sig := range t.multiSigTopology.GetSigners() {
		signers[sig] = struct{}{}
	}

	// if there are not enough slots, demote from tm to remaining
	tendermintValidators, remainingValidators, removedFromTM := handleSlotChanges(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, t.numberOfTendermintValidators, int64(t.currentBlockHeight+1), rankingScore, signers, t.multiSigTopology.GetThreshold())
	t.log.Info("removedFromTM", logging.Strings("IDs", removedFromTM))

	// now we're sorting the remaining validators - some of which may be eratz, some may have been tendermint (as demoted above) and some just in the waiting list
	// we also sort the tendermint set again as there may have been a promotion due to a slot change
	sortValidatorDescRankingScoreAscBlockcompare(tendermintValidators, rankingScore, byStatusChangeBlock, t.rng)
	sortValidatorDescRankingScoreAscBlockcompare(remainingValidators, rankingScore, byBlockAdded, t.rng)

	// apply promotions and demotions from tendermint to ersatz and vice versa
	remainingValidators, demotedFromTM := promote(tendermintValidators, remainingValidators, ValidatorStatusTendermint, ValidatorStatusErsatz, t.numberOfTendermintValidators, rankingScore, int64(t.currentBlockHeight+1))
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

	// demoted tendermint validators are also ersatz so we need to add them
	for _, id := range demotedFromTM {
		ersatzValidators = append(ersatzValidators, t.validators[id])
	}

	// demote from ersatz to pending due to more ersatz than slots allowed
	ersatzValidators, waitingListValidators, _ = handleSlotChanges(ersatzValidators, waitingListValidators, ValidatorStatusErsatz, ValidatorStatusPending, t.numberOfErsatzValidators, int64(t.currentBlockHeight+1), rankingScore, map[string]struct{}{}, 0)
	sortValidatorDescRankingScoreAscBlockcompare(ersatzValidators, rankingScore, byBlockAdded, t.rng)
	sortValidatorDescRankingScoreAscBlockcompare(waitingListValidators, rankingScore, byBlockAdded, t.rng)
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
	// optimal stake is calculated with respect to tendermint validators, i.e. total stake by tendermint validators
	optimalStake := GetOptimalStake(tmTotalDelegation, len(tmDelegation), stakeScoreParams)

	// calculate the anti-whaling stake score of the validators with respect to stake represented by the tm validators
	nextValidatorsStakeScore := CalcAntiWhalingScore(tmDelegation, tmTotalDelegation, optimalStake, stakeScoreParams)

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
		pkey := vu.PubKey.GetEd25519()
		if pkey == nil || len(pkey) <= 0 {
			pkey = vu.PubKey.GetSecp256K1()
		}
		// tendermint pubkey are marshalled in base64,
		// so let's do this as well here for logging
		spkey := base64.StdEncoding.EncodeToString(pkey)

		t.log.Info("voting power update", logging.String("pubKey", spkey), logging.Int64("power", vu.Power))
	}

	return vUpdates, nextValidatorsVotingPower
}

// handleSlotChanges the number of slots may have increased or decreased and so we slide the nodes into the different sets based on the change.
func handleSlotChanges(seriesA []*valState, seriesB []*valState, statusA ValidatorStatus, statusB ValidatorStatus, maxForSeriesA int, nextBlockHeight int64, rankingScore map[string]num.Decimal, signers map[string]struct{}, multisigThreshold uint32) ([]*valState, []*valState, []string) {
	removedFromSeriesA := []string{}

	if len(seriesA) == maxForSeriesA {
		// no change we're done
		return seriesA, seriesB, removedFromSeriesA
	}

	removed := 0
	removedSigners := 0

	// count how many signers we have in the validtor set - we need to do that as the contract may not have been updated with the signers yet but the
	// list of validators has been updated.
	numSigners := 0
	for _, vs := range seriesA {
		if _, ok := signers[vs.data.EthereumAddress]; ok {
			numSigners++
		}
	}

	// the number of slots for series A has decrease, move some into series B
	// when demoting from tendermint - we only allow removal of one signer per round as long as there are sufficient validators remaining
	// to satisfy the threshold.
	if len(seriesA) > maxForSeriesA {
		nDescreased := len(seriesA) - maxForSeriesA
		for i := 0; i < nDescreased; i++ {
			toDemote := seriesA[len(seriesA)-1-i]
			if _, ok := signers[toDemote.data.EthereumAddress]; ok {
				if len(signers) > 0 && uint32(1000*(numSigners-1)/len(signers)) <= multisigThreshold {
					break
				}
				removed++
				removedSigners++
			} else {
				removed++
			}

			toDemote.status = statusB
			toDemote.statusChangeBlock = nextBlockHeight

			// add to the remaining validators so it can compete with the ersatzvalidators
			seriesB = append(seriesB, toDemote)
			removedFromSeriesA = append(removedFromSeriesA, toDemote.data.ID)
			if removedSigners > 0 {
				break
			}
		}

		// they've been added to seriesB slice, remove them from seriesA
		seriesA = seriesA[:len(seriesA)-removed]
		return seriesA, seriesB, removedFromSeriesA
	}

	// the number of slots for series A has increased, move some in from series B
	if len(seriesA) < maxForSeriesA && len(seriesB) > 0 {
		nIncreased := maxForSeriesA - len(seriesA)

		if nIncreased > len(seriesB) {
			nIncreased = len(seriesB)
		}

		for i := 0; i < nIncreased; i++ {
			toPromote := seriesB[0]

			score := rankingScore[toPromote.data.ID]
			if score.IsZero() {
				break // the nodes are ordered by ranking score and we do not want to promote one with 0 score so we stop here
			}

			toPromote.status = statusA
			toPromote.statusChangeBlock = nextBlockHeight
			// add to the remaining validators so it can compete with the ersatzvalidators
			seriesA = append(seriesA, toPromote)
			seriesB = seriesB[1:]
		}
		return seriesA, seriesB, removedFromSeriesA
	}

	return seriesA, seriesB, removedFromSeriesA
}

// promote returns seriesA and seriesB updated with promotions moved from B to A and a slice of removed from series A in case of swap-promotion.
func promote(seriesA []*valState, seriesB []*valState, statusA ValidatorStatus, statusB ValidatorStatus, maxForSeriesA int, rankingScore map[string]num.Decimal, nextBlockHeight int64) ([]*valState, []string) {
	removedFromSeriesA := []string{}

	if maxForSeriesA > 0 && maxForSeriesA == len(seriesA) && len(seriesB) > 0 {
		// the best of the remaining is better than the worst tendermint validator
		if rankingScore[seriesA[len(seriesA)-1].data.ID].LessThan(rankingScore[seriesB[0].data.ID]) {
			vd := seriesA[len(seriesA)-1]
			vd.status = statusB
			vd.statusChangeBlock = nextBlockHeight
			removedFromSeriesA = append(removedFromSeriesA, vd.data.ID)

			vd = seriesB[0]
			vd.status = statusA
			vd.statusChangeBlock = nextBlockHeight
			if len(seriesB) > 1 {
				seriesB = seriesB[1:]
			} else {
				seriesB = []*valState{}
			}
		}
	}
	return seriesB, removedFromSeriesA
}

// calculateVotingPower returns the voting powers as the normalised ranking scores scaled by VotingPowerScalingFactor with a minimum of 1.
func (t *Topology) calculateVotingPower(IDs []string, rankingScores map[string]num.Decimal) map[string]int64 {
	votingPower := make(map[string]int64, len(IDs))
	sumOfScores := num.DecimalZero()
	for _, ID := range IDs {
		sumOfScores = sumOfScores.Add(rankingScores[ID])
	}

	for _, ID := range IDs {
		if sumOfScores.IsPositive() {
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
		// it's safer to reset the validator performance counter here which is the exact time we're updating tendermint on the voting power.
		t.validatorPerformance.Reset()
		return t.validatorPowerUpdates
	}
	return []tmtypes.ValidatorUpdate{}
}
