package subscribers

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega/events/v1"
)

type RE interface {
	events.Event
	RewardPayoutEvent() types.RewardPayoutEvent
}

// RewardDetails holds all the details about a single asset based reward
type RewardDetails struct {
	// The type of asset this reward is for
	AssetID string
	// The total amount of reward received
	TotalReward uint64
	// Last reward amount
	LastReward uint64
	// Last percentage amount
	LastPercentageAmount float64
}

// RewardCounters hold the details of all the different rewards for each party
type RewardCounters struct {
	*Base
	mu sync.RWMutex

	// Map of partyID to reward details
	rewards map[string]*RewardDetails

	// Logger
	log *logging.Logger
}

// NewRewards constructor to create an object to handle reward totals
func NewRewards(ctx context.Context, log *logging.Logger, ack bool) *RewardCounters {
	rc := RewardCounters{
		Base:    NewBase(ctx, 10, ack),
		log:     log,
		rewards: map[string]*RewardDetails{},
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
			tr := et.RewardPayoutEvent()
			rc.updateRewards(tr)
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

func (rc *RewardCounters) UpdateRewards(rpe types.RewardPayoutEvent) {
	rc.updateRewards(rpe)
}

func (rc *RewardCounters) updateRewards(rpe types.RewardPayoutEvent) {
	percentage, err := strconv.ParseFloat(rpe.PercentOfTotalReward, 64)
	if err != nil {
		percentage = 0.0
	}

	rc.mu.RLock()
	defer rc.mu.RUnlock()

	reward, ok := rc.rewards[rpe.Party]

	if !ok {
		// First reward for this party
		reward := &RewardDetails{
			AssetID:              rpe.Asset,
			TotalReward:          rpe.Amount,
			LastReward:           rpe.Amount,
			LastPercentageAmount: percentage,
		}
		rc.rewards[rpe.Party] = reward
		return
	}

	reward.LastReward = rpe.Amount
	reward.TotalReward += rpe.Amount
	reward.LastPercentageAmount = percentage
}

// GetRewardDetails returns the information relating to rewards for a single party
func (rc *RewardCounters) GetRewardDetails(ctx context.Context, partyID string) (*RewardDetails, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	reward, ok := rc.rewards[partyID]
	if !ok {
		return nil, fmt.Errorf("No rewards found for partyID %s", partyID)
	}
	// Create a copy and return
	rewardCopy := *reward
	return &rewardCopy, nil
}
