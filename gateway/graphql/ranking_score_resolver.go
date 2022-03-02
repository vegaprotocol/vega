package gql

import (
	"context"
	"strconv"

	proto "code.vegaprotocol.io/protos/vega"
)

type rankingScoreResolver VegaResolverRoot
type rewardScoreResolver VegaResolverRoot

func convertValidatorStatusFromProto(status proto.ValidatorNodeStatus) string {
	if status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ {
		return "ersatz"
	}
	if status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT {
		return "tendermint"
	}
	if status == proto.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_PENDING {
		return "pending"
	}
	return "unspecified"
}

func (r *rankingScoreResolver) PreviousStatus(ctx context.Context, obj *proto.RankingScore) (string, error) {
	return convertValidatorStatusFromProto(obj.PreviousStatus), nil
}

func (r *rankingScoreResolver) Status(ctx context.Context, obj *proto.RankingScore) (string, error) {
	return convertValidatorStatusFromProto(obj.Status), nil
}

func (r *rankingScoreResolver) VotingPower(ctx context.Context, obj *proto.RankingScore) (string, error) {
	return strconv.FormatUint(uint64(obj.VotingPower), 10), nil
}

func (t *rewardScoreResolver) ValidatorStatus(ctx context.Context, obj *proto.RewardScore) (string, error) {
	return convertValidatorStatusFromProto(obj.ValidatorStatus), nil
}
