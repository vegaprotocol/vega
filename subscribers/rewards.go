package subscribers

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/protos/vega"
	types "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types/num"
)

type RE interface {
	events.Event
	RewardPayoutEvent() types.RewardPayoutEvent
}

// rewardDetails holds all the details about a single asset based reward
type rewardDetails struct {
	// The asset this reward is for
	assetID string
	// The party that received the reward
	partyID string
	// Which epoch this reward was calculated
	epoch uint64
	// The total amount of reward received
	amount *num.Uint
	// Percentage of total reward distributed
	percentageAmount float64
	// When the reward was received
	receivedAt int64
}

// rewardsPerAssetDetails contains all rewards received per asset
type rewardsPerAssetDetails struct {
	// The asset this reward is for
	asset string
	// Slice containing all rewards we have received
	rewards []*rewardDetails
	// Total amount of reward received
	totalAmount *num.Uint
}

type rewardsPerPartyDetails struct {
	// The party that received the reward
	partyID string
	// Map of assetID to PerAsset type
	rewards map[string]*rewardsPerAssetDetails
}

// RewardCounters hold the details of all the different rewards for each party
type RewardCounters struct {
	*Base

	// Map of partyID to reward details
	rewards map[string]*rewardsPerPartyDetails
	mu      sync.RWMutex

	// Logger
	log *logging.Logger
}

// NewRewards constructor to create an object to handle reward totals
func NewRewards(ctx context.Context, log *logging.Logger, ack bool) *RewardCounters {
	rc := RewardCounters{
		Base:    NewBase(ctx, 10, ack),
		log:     log,
		rewards: map[string]*rewardsPerPartyDetails{},
	}

	if rc.isRunning() {
		go rc.loop(rc.ctx)
	}
	return &rc
}

func (rc *RewardCounters) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rc.Halt()
			return
		case e := <-rc.ch:
			if rc.isRunning() {
				rc.Push(e...)
			}
		}
	}
}

// Push takes transfer request messages and uses them to update the rewards totals
func (rc *RewardCounters) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {
		case RE:
			rc.updateRewards(et.RewardPayoutEvent())
		default:
			rc.log.Panic("Unknown event type in reward counters", logging.String("Type", et.Type().String()))
		}
	}
}

// Types returns all the message types this subscriber wants to receive
func (rc *RewardCounters) Types() []events.Type {
	return []events.Type{
		events.TransferResponses,
	}
}

func (rc *RewardCounters) addNewReward(perParty *rewardsPerPartyDetails, rpe types.RewardPayoutEvent) {
	perAsset, ok := perParty.rewards[rpe.Asset]
	if !ok {
		// First reward for this asset
		perAsset = &rewardsPerAssetDetails{
			asset:       rpe.Asset,
			rewards:     make([]*rewardDetails, 0),
			totalAmount: num.Zero(),
		}
		perParty.rewards[rpe.Asset] = perAsset
	}

	epoch, err := strconv.ParseUint(rpe.EpochSeq, 10, 64)
	if err != nil {
		epoch = 0
	}

	percent, err := strconv.ParseFloat(rpe.PercentOfTotalReward, 64)
	if err != nil {
		percent = 0.0
	}

	amount, _ := num.UintFromString(rpe.Amount, 10)
	rd := &rewardDetails{
		assetID:          rpe.Asset,
		partyID:          rpe.Party,
		epoch:            epoch,
		amount:           amount,
		percentageAmount: percent,
		receivedAt:       0,
	}

	perAsset.rewards = append(perAsset.rewards, rd)
	perAsset.totalAmount.AddSum(rd.amount)
}

func (rc *RewardCounters) updateRewards(rpe types.RewardPayoutEvent) {
	rc.mu.RLock()
	reward, ok := rc.rewards[rpe.Party]
	rc.mu.RUnlock()

	if !ok {
		// First reward for this party
		reward = &rewardsPerPartyDetails{
			partyID: rpe.Party,
			rewards: map[string]*rewardsPerAssetDetails{},
		}
		rc.mu.Lock()
		rc.rewards[rpe.Party] = reward
		rc.mu.Unlock()
	}
	rc.addNewReward(reward, rpe)
}

// GetRewardDetails returns the information relating to rewards for a single party
func (rc *RewardCounters) GetRewardDetails(ctx context.Context, partyID string) (*protoapi.GetRewardDetailsResponse, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	rewards, ok := rc.rewards[partyID]
	if !ok {
		return nil, fmt.Errorf("no rewards found for partyid %s", partyID)
	}

	// Now build up the proto message from the details we have stored
	resp := &protoapi.GetRewardDetailsResponse{
		RewardDetails: make([]*vega.RewardPerAssetDetail, 0),
	}

	for _, rpad := range rewards.rewards {
		perAsset := vega.RewardPerAssetDetail{
			Asset:         rpad.asset,
			TotalForAsset: rpad.totalAmount.String(),
			Details:       make([]*vega.RewardDetails, 0),
		}
		for _, rd := range rpad.rewards {
			reward := vega.RewardDetails{
				AssetId:           rd.assetID,
				PartyId:           rd.partyID,
				Epoch:             rd.epoch,
				Amount:            rd.amount.String(),
				PercentageOfTotal: strconv.FormatFloat(rd.percentageAmount, 'f', 5, 64),
				ReceivedAt:        rd.receivedAt,
			}
			perAsset.Details = append(perAsset.Details, &reward)
		}
		resp.RewardDetails = append(resp.RewardDetails, &perAsset)
	}
	return resp, nil
}
