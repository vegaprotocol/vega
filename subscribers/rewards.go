package subscribers

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/data-node/contextutil"
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

// rewardsDetails contains all rewards
type rewardsDetails struct {
	// Slice containing all rewards we have received
	rewards []*rewardDetails
	// Total amount of reward received
	totalAmount *num.Uint
}

// RewardCounters hold the details of all the different rewards for each party
type RewardCounters struct {
	*Base

	// Map of map per partyID per asset to reward details
	rewardsPerPartyPerAsset map[string]map[string]*rewardsDetails
	mu                      sync.RWMutex

	subscriberCnt int32
	subscribers   map[uint64]subscription
	subscriberID  uint64

	// Logger
	log *logging.Logger
}

type rewardFilter struct {
	assetID string
	party   string
}

func (rf rewardFilter) filter(rw vega.RewardDetails) bool {
	return (len(rf.assetID) <= 0 || rf.assetID == rw.AssetId) && (len(rf.party) <= 0 || rf.party == rw.PartyId)
}

type subscription struct {
	subscriber chan vega.RewardDetails
	filter     rewardFilter
	cancel     func()
	retries    int
}

// NewRewards constructor to create an object to handle reward totals
func NewRewards(ctx context.Context, log *logging.Logger, ack bool) *RewardCounters {
	rc := RewardCounters{
		Base:                    NewBase(ctx, 10, ack),
		log:                     log,
		rewardsPerPartyPerAsset: map[string]map[string]*rewardsDetails{},
		subscribers:             map[uint64]subscription{},
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
			rc.addNewReward(et.RewardPayoutEvent())
		default:
			rc.log.Panic("Unknown event type in reward counters", logging.String("type", et.Type().String()))
		}
	}
}

func (rc *RewardCounters) addNewReward(rpe types.RewardPayoutEvent) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if _, ok := rc.rewardsPerPartyPerAsset[rpe.Party]; !ok {
		rc.rewardsPerPartyPerAsset[rpe.Party] = map[string]*rewardsDetails{}
	}

	if _, ok := rc.rewardsPerPartyPerAsset[rpe.Party][rpe.Asset]; !ok {
		// First reward for this asset
		perAsset := &rewardsDetails{
			rewards:     make([]*rewardDetails, 0),
			totalAmount: num.Zero(),
		}
		rc.rewardsPerPartyPerAsset[rpe.Party][rpe.Asset] = perAsset
	}

	perAsset := rc.rewardsPerPartyPerAsset[rpe.Party][rpe.Asset]

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
		receivedAt:       rpe.Timestamp,
	}

	perAsset.rewards = append(perAsset.rewards, rd)
	perAsset.totalAmount.AddSum(rd.amount)

	rc.notifyWithLock(vega.RewardDetails{
		AssetId:           rd.assetID,
		PartyId:           rd.partyID,
		Epoch:             rd.epoch,
		Amount:            rd.amount.String(),
		PercentageOfTotal: strconv.FormatFloat(rd.percentageAmount, 'f', 5, 64),
		ReceivedAt:        rd.receivedAt,
	})
}

func (rc *RewardCounters) notifyWithLock(rd vega.RewardDetails) {
	if len(rc.subscribers) == 0 {
		return
	}

	for id, sub := range rc.subscribers {
		if sub.filter.filter(rd) {
			retryCount := sub.retries
			ok := false
			for !ok && retryCount >= 0 {
				select {
				case sub.subscriber <- rd:
					rc.log.Debug(
						"Reward details for subscriber sent successfully",
						logging.Uint64("ref", id),
					)
					ok = true
				default:
					retryCount--
					if retryCount > 0 {
						rc.log.Debug(
							"Reward details for subscriber not sent",
							logging.Uint64("ref", id))
					}
					time.Sleep(time.Duration(10) * time.Millisecond)
				}
			}
			if !ok && retryCount <= 0 {
				rc.log.Warn(
					"Reward details subscriber has hit the retry limit",
					logging.Uint64("ref", id),
					logging.Int("retries", sub.retries))
				sub.cancel()
			}
		}
	}
}

//subscribe allows a client to register for updates of the reward details.
func (rc *RewardCounters) subscribe(sub subscription) uint64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.subscriberID++
	rc.subscribers[rc.subscriberID] = sub

	rc.log.Debug("reward details subscriber added store",
		logging.Uint64("subscriber-id", rc.subscriberID))

	return rc.subscriberID
}

// Unsubscribe allows the client to unregister interest in reward details.
func (rc *RewardCounters) unsubscribe(id uint64) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if len(rc.subscribers) == 0 {
		rc.log.Debug("Un-subscribe called in reward details, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := rc.subscribers[id]; exists {
		delete(rc.subscribers, id)
		return nil
	}

	return fmt.Errorf("subscriber to delegation updates does not exist with id: %d", id)
}

//ObserveRewardDetails returns a channel for subscribing to reward details.
func (rc *RewardCounters) ObserveRewardDetails(ctx context.Context, retries int, assetID, party string) (rewardCh <-chan vega.RewardDetails, ref uint64) {
	rewards := make(chan vega.RewardDetails)
	ctx, cancel := context.WithCancel(ctx)
	ref = rc.subscribe(subscription{
		filter: rewardFilter{
			assetID: assetID,
			party:   party,
		},
		subscriber: rewards,
		cancel:     cancel,
		retries:    retries,
	})

	go func() {
		atomic.AddInt32(&rc.subscriberCnt, 1)
		defer atomic.AddInt32(&rc.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for range ctx.Done() {
			rc.log.Debug(
				"rewards subscriber closed connection",
				logging.Uint64("id", ref),
				logging.String("ip-address", ip),
			)
			// this error only happens when the subscriber reference doesn't exist
			// so we can still safely close the channels
			if err := rc.unsubscribe(ref); err != nil {
				rc.log.Error(
					"Failure un-subscribing delegations subscriber when context.Done()",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
					logging.Error(err),
				)
			}
			close(rewards)
			return
		}
	}()

	return rewards, ref
}

// GetRewardSubscribersCount returns the total number of active subscribers for ObserveRewardDetails.
func (rc *RewardCounters) GetRewardSubscribersCount() int32 {
	return atomic.LoadInt32(&rc.subscriberCnt)
}

// GetRewardDetails returns the information relating to rewards for a single party
func (rc *RewardCounters) GetRewardDetails(ctx context.Context, partyID string) (*protoapi.GetRewardDetailsResponse, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	rewards, ok := rc.rewardsPerPartyPerAsset[partyID]
	if !ok {
		rewards = make(map[string]*rewardsDetails)
	}

	// Now build up the proto message from the details we have stored
	resp := &protoapi.GetRewardDetailsResponse{
		RewardDetails: make([]*vega.RewardPerAssetDetail, 0, len(rewards)),
	}

	for asset, rpad := range rewards {
		perAsset := vega.RewardPerAssetDetail{
			Asset:         asset,
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

// Types returns all the message types this subscriber wants to receive
func (rc *RewardCounters) Types() []events.Type {
	return []events.Type{
		events.RewardPayoutEvent,
	}
}
