package entities

import (
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
)

type Reward struct {
	PartyID        PartyID
	AssetID        AssetID
	MarketID       MarketID
	EpochID        int64
	Amount         decimal.Decimal
	PercentOfTotal float64
	RewardType     string
	Timestamp      time.Time
	VegaTime       time.Time
}

func (r Reward) String() string {
	return fmt.Sprintf("{Epoch: %v, Party: %s, Asset: %s, Amount: %v}",
		r.EpochID, r.PartyID, r.AssetID, r.Amount)
}

func (r *Reward) ToProto() *vega.Reward {
	protoReward := vega.Reward{
		PartyId:           r.PartyID.String(),
		AssetId:           r.AssetID.String(),
		Epoch:             uint64(r.EpochID),
		Amount:            r.Amount.String(),
		PercentageOfTotal: fmt.Sprintf("%v", r.PercentOfTotal),
		ReceivedAt:        r.Timestamp.UnixNano(),
		MarketId:          r.MarketID.String(),
		RewardType:        r.RewardType,
	}
	return &protoReward
}

func RewardFromProto(pr eventspb.RewardPayoutEvent, vegaTime time.Time) (Reward, error) {
	epochID, err := strconv.ParseInt(pr.EpochSeq, 10, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing epoch '%v': %w", pr.EpochSeq, err)
	}

	percentOfTotal, err := strconv.ParseFloat(pr.PercentOfTotalReward, 64)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing percent of total reward '%v': %w",
			pr.PercentOfTotalReward, err)
	}

	amount, err := decimal.NewFromString(pr.Amount)
	if err != nil {
		return Reward{}, fmt.Errorf("parsing amount of reward: '%v': %w",
			pr.Amount, err)
	}

	reward := Reward{
		PartyID:        NewPartyID(pr.Party),
		AssetID:        NewAssetID(pr.Asset),
		EpochID:        epochID,
		Amount:         amount,
		PercentOfTotal: percentOfTotal,
		Timestamp:      NanosToPostgresTimestamp(pr.Timestamp),
		MarketID:       NewMarketID(pr.Market),
		RewardType:     pr.RewardType,
		VegaTime:       vegaTime,
	}

	return reward, nil
}
