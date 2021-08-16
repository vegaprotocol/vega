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
<<<<<<< HEAD
	"code.vegaprotocol.io/vega/types/num"
=======
>>>>>>> get event code from vega repo
)

type RE interface {
	events.Event
	RewardPayoutEvent() types.RewardPayoutEvent
}

// RewardDetails holds all the details about a single asset based reward
type RewardDetails struct {
	// The asset this reward is for
	AssetID string
	// The party that received the reward
	PartyID string
	// Which epoch this reward was calculated
	Epoch uint64
	// The total amount of reward received
	Amount *num.Uint
	// Percentage of total reward distributed
	PercentageAmount float64
	// When the reward was received
	ReceivedAt int64
}

// RewardPerAssetDetails contains all rewards received per asset
type RewardsPerAssetDetails struct {
	// The asset this reward is for
	Asset string
	// Slice containing all rewards we have received
	Rewards []*RewardDetails
	// Total amount of reward received
	TotalAmount *num.Uint
}

type RewardsPerPartyDetails struct {
	// The party that received the reward
	PartyID string
	// Map of partyID to PerAsset type
	Rewards map[string]*RewardsPerAssetDetails
}

// RewardCounters hold the details of all the different rewards for each party
type RewardCounters struct {
	*Base
	mu sync.RWMutex

	// Map of partyID to reward details
	rewards map[string]*RewardsPerPartyDetails

	// Logger
	log *logging.Logger
}

// NewRewards constructor to create an object to handle reward totals
func NewRewards(ctx context.Context, log *logging.Logger, ack bool) *RewardCounters {
	rc := RewardCounters{
		Base:    NewBase(ctx, 10, ack),
		log:     log,
		rewards: map[string]*RewardsPerPartyDetails{},
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

func (rc *RewardCounters) addNewReward(perParty *RewardsPerPartyDetails, rpe types.RewardPayoutEvent) {
	perAsset, ok := perParty.Rewards[rpe.Asset]
	if !ok {
		// First reward for this asset
		perAsset = &RewardsPerAssetDetails{
			Asset:       rpe.Asset,
			Rewards:     make([]*RewardDetails, 0),
			TotalAmount: num.Zero(),
		}
		perParty.Rewards[rpe.Asset] = perAsset
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
	rd := &RewardDetails{
		AssetID:          rpe.Asset,
		PartyID:          rpe.Party,
		Epoch:            epoch,
		Amount:           amount,
		PercentageAmount: percent,
		ReceivedAt:       0,
	}

	perAsset.Rewards = append(perAsset.Rewards, rd)
	perAsset.TotalAmount.AddSum(rd.Amount)
}

func (rc *RewardCounters) updateRewards(rpe types.RewardPayoutEvent) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	reward, ok := rc.rewards[rpe.Party]

	if !ok {
		// First reward for this party
		reward = &RewardsPerPartyDetails{
			PartyID: rpe.Party,
			Rewards: map[string]*RewardsPerAssetDetails{},
		}
		rc.rewards[rpe.Party] = reward
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

	for _, rpad := range rewards.Rewards {
		perAsset := vega.RewardPerAssetDetail{
			Asset:         rpad.Asset,
			TotalForAsset: rpad.TotalAmount.String(),
			Details:       make([]*vega.RewardDetails, 0),
		}
		for _, rd := range rpad.Rewards {
			reward := vega.RewardDetails{
				AssetId:           rd.AssetID,
				PartyId:           rd.PartyID,
				Epoch:             rd.Epoch,
				Amount:            rd.Amount.String(),
				PercentageOfTotal: strconv.FormatFloat(rd.PercentageAmount, 'f', 5, 64),
				ReceivedAt:        rd.ReceivedAt,
			}
			perAsset.Details = append(perAsset.Details, &reward)
		}
		resp.RewardDetails = append(resp.RewardDetails, &perAsset)
	}
	return resp, nil
}
