// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"strconv"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
)

type NodeID struct{ ID }

func NewNodeID(id string) NodeID {
	return NodeID{ID: ID(id)}
}

type Node struct {
	ID                NodeID
	PubKey            VegaPublicKey       `db:"vega_pub_key"`
	TmPubKey          TendermintPublicKey `db:"tendermint_pub_key"`
	EthereumAddress   EthereumAddress
	InfoUrl           string
	Location          string
	StakedByOperator  decimal.Decimal
	StakedByDelegates decimal.Decimal
	StakedTotal       decimal.Decimal
	MaxIntendedStake  decimal.Decimal
	PendingStake      decimal.Decimal
	EpochData         EpochData
	Status            NodeStatus
	Delegations       []Delegation  `json:""`
	RewardScore       *RewardScore  `json:""`
	RankingScore      *RankingScore `json:""`
	Name              string
	AvatarUrl         string
	VegaTime          time.Time
}

type ValidatorUpdateAux struct {
	Added           bool
	FromEpoch       uint64
	VegaPubKeyIndex uint32
}

type EpochData struct {
	*vega.EpochData
}

type RewardScore struct {
	RawValidatorScore   decimal.Decimal     `json:"raw_validator_score"`
	PerformanceScore    decimal.Decimal     `json:"performance_score"`
	MultisigScore       decimal.Decimal     `json:"multisig_score"`
	ValidatorScore      decimal.Decimal     `json:"validator_score"`
	NormalisedScore     decimal.Decimal     `json:"normalised_score"`
	ValidatorNodeStatus ValidatorNodeStatus `json:"validator_node_status,string"`
	VegaTime            time.Time           `json:"vega_time"`
	EpochSeq            uint64
}

type RewardScoreAux struct {
	NodeId   NodeID
	EpochSeq uint64
}

type RankingScore struct {
	StakeScore       decimal.Decimal     `json:"stake_score"`
	PerformanceScore decimal.Decimal     `json:"performance_score"`
	PreviousStatus   ValidatorNodeStatus `json:"previous_status,string"`
	Status           ValidatorNodeStatus `json:",string"`
	VotingPower      uint32              `json:"voting_power"`
	RankingScore     decimal.Decimal     `json:"ranking_score"`
	VegaTime         time.Time           `json:"vega_time"`
	EpochSeq         uint64
}

type RankingScoreAux struct {
	NodeId   NodeID
	EpochSeq uint64
}

type NodeData struct {
	StakedTotal     decimal.Decimal
	TotalNodes      uint32
	InactiveNodes   uint32
	ValidatingNodes uint32
	Uptime          float64
	VegaTime        time.Time
}

func NodeFromValidatorUpdateEvent(evt eventspb.ValidatorUpdate, vegaTime time.Time) (Node, ValidatorUpdateAux, error) {
	return Node{
			ID:              NewNodeID(evt.NodeId),
			PubKey:          VegaPublicKey(evt.VegaPubKey),
			TmPubKey:        TendermintPublicKey(evt.TmPubKey),
			EthereumAddress: EthereumAddress(evt.EthereumAddress),
			InfoUrl:         evt.InfoUrl,
			Location:        evt.Country,
			Name:            evt.Name,
			AvatarUrl:       evt.AvatarUrl,
			VegaTime:        vegaTime,

			// Not present in the event
			Status:            NodeStatusValidator, // This was the default value in the legacy store code
			StakedByOperator:  decimal.Zero,
			StakedByDelegates: decimal.Zero,
			StakedTotal:       decimal.Zero,
			MaxIntendedStake:  decimal.Zero,
			PendingStake:      decimal.Zero,
			EpochData:         EpochData{},
			Delegations:       []Delegation{},
			RewardScore:       nil,
			RankingScore:      nil,
		}, ValidatorUpdateAux{
			Added:           evt.Added,
			FromEpoch:       evt.FromEpoch,
			VegaPubKeyIndex: evt.VegaPubKeyIndex,
		}, nil
}

func ValidatorNodeStatusFromString(status string) ValidatorNodeStatus {
	switch status {
	case "tendermint":
		return ValidatorNodeStatusTendermint
	case "ersatz":
		return ValidatorNodeStatusErsatz
	case "pending":
		return ValidatorNodeStatusPending
	case "unspecified":
		fallthrough
	default: // Is this appropriate behaviour? Should we error on the default case?
		return ValidatorNodeStatusUnspecified
	}
}

func RankingScoreFromRankingEvent(evt eventspb.ValidatorRankingEvent, vegaTime time.Time) (RankingScore, RankingScoreAux, error) {
	stakeScore, err := decimal.NewFromString(evt.StakeScore)
	if err != nil {
		return RankingScore{}, RankingScoreAux{}, err
	}

	performanceScore, err := decimal.NewFromString(evt.PerformanceScore)
	if err != nil {
		return RankingScore{}, RankingScoreAux{}, err
	}

	rankingScore, err := decimal.NewFromString(evt.RankingScore)
	if err != nil {
		return RankingScore{}, RankingScoreAux{}, err
	}

	epochSeq, err := strconv.ParseUint(evt.EpochSeq, 10, 64)
	if err != nil {
		return RankingScore{}, RankingScoreAux{}, err
	}

	return RankingScore{
			StakeScore:       stakeScore,
			PerformanceScore: performanceScore,
			PreviousStatus:   ValidatorNodeStatusFromString(evt.PreviousStatus),
			Status:           ValidatorNodeStatusFromString(evt.NextStatus),
			VotingPower:      evt.TmVotingPower,
			RankingScore:     rankingScore,
			VegaTime:         vegaTime,
			EpochSeq:         epochSeq,
		}, RankingScoreAux{
			NodeId:   NewNodeID(evt.NodeId),
			EpochSeq: epochSeq,
		}, nil
}

func (rs *RankingScore) ToProto() *vega.RankingScore {
	return &vega.RankingScore{
		StakeScore:       rs.StakeScore.String(),
		PerformanceScore: rs.PerformanceScore.String(),
		PreviousStatus:   vega.ValidatorNodeStatus(rs.PreviousStatus),
		Status:           vega.ValidatorNodeStatus(rs.Status),
		VotingPower:      rs.VotingPower,
		RankingScore:     rs.RankingScore.String(),
	}
}

func RewardScoreFromScoreEvent(evt eventspb.ValidatorScoreEvent, vegaTime time.Time) (RewardScore, RewardScoreAux, error) {
	rawValidatorScore, err := decimal.NewFromString(evt.RawValidatorScore)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	performanceScore, err := decimal.NewFromString(evt.ValidatorPerformance)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	multisigScore, err := decimal.NewFromString(evt.MultisigScore)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	validatorScore, err := decimal.NewFromString(evt.ValidatorScore)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	normalisedScore, err := decimal.NewFromString(evt.NormalisedScore)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	epochSeq, err := strconv.ParseUint(evt.EpochSeq, 10, 64)
	if err != nil {
		return RewardScore{}, RewardScoreAux{}, err
	}

	return RewardScore{
			RawValidatorScore:   rawValidatorScore,
			PerformanceScore:    performanceScore,
			MultisigScore:       multisigScore,
			ValidatorScore:      validatorScore,
			NormalisedScore:     normalisedScore,
			ValidatorNodeStatus: ValidatorNodeStatusFromString(evt.ValidatorStatus),
			VegaTime:            vegaTime,
			EpochSeq:            epochSeq,
		}, RewardScoreAux{
			NodeId:   NewNodeID(evt.NodeId),
			EpochSeq: epochSeq,
		}, nil
}

func (rs *RewardScore) ToProto() *vega.RewardScore {
	return &vega.RewardScore{
		RawValidatorScore: rs.RawValidatorScore.String(),
		PerformanceScore:  rs.PerformanceScore.String(),
		MultisigScore:     rs.MultisigScore.String(),
		ValidatorScore:    rs.ValidatorScore.String(),
		NormalisedScore:   rs.NormalisedScore.String(),
		ValidatorStatus:   vega.ValidatorNodeStatus(rs.ValidatorNodeStatus),
	}
}

func NodeFromProto(node *vega.Node, vegaTime time.Time) (Node, error) {
	stakedByOperator, err := decimal.NewFromString(node.StakedByOperator)
	if err != nil {
		return Node{}, err
	}

	stakedByDelegates, err := decimal.NewFromString(node.StakedByDelegates)
	if err != nil {
		return Node{}, err
	}

	stakedTotal, err := decimal.NewFromString(node.StakedTotal)
	if err != nil {
		return Node{}, err
	}

	maxIntendedStake, err := decimal.NewFromString(node.MaxIntendedStake)
	if err != nil {
		return Node{}, err
	}

	pendingStake, err := decimal.NewFromString(node.PendingStake)
	if err != nil {
		return Node{}, err
	}

	delegations := make([]Delegation, len(node.Delegations))
	for i, delegation := range node.Delegations {
		delegations[i], err = DelegationFromProto(delegation)
		if err != nil {
			return Node{}, err
		}
	}

	return Node{
		ID:                NewNodeID(node.Id),
		PubKey:            VegaPublicKey(node.PubKey),
		TmPubKey:          TendermintPublicKey(node.TmPubKey),
		EthereumAddress:   EthereumAddress(node.EthereumAdddress),
		InfoUrl:           node.InfoUrl,
		Location:          node.Location,
		StakedByOperator:  stakedByOperator,
		StakedByDelegates: stakedByDelegates,
		StakedTotal:       stakedTotal,
		MaxIntendedStake:  maxIntendedStake,
		PendingStake:      pendingStake,
		EpochData:         EpochData{node.EpochData},
		Status:            NodeStatus(node.Status),
		Delegations:       delegations,
		// RewardScore:       RewardScore{node.RewardScore},
		// RankingScore:      RankingScore{node.RankingScore},
		Name:      node.Name,
		AvatarUrl: node.AvatarUrl,
		VegaTime:  vegaTime,
	}, nil
}

func (node *Node) ToProto() *vega.Node {
	protoDelegations := make([]*vega.Delegation, len(node.Delegations))
	for i, delegation := range node.Delegations {
		protoDelegations[i] = delegation.ToProto()
	}

	res := &vega.Node{
		Id:                node.ID.String(),
		PubKey:            node.PubKey.String(),
		TmPubKey:          node.TmPubKey.String(),
		EthereumAdddress:  node.EthereumAddress.String(),
		InfoUrl:           node.InfoUrl,
		Location:          node.Location,
		StakedByOperator:  node.StakedByOperator.String(),
		StakedByDelegates: node.StakedByDelegates.String(),
		StakedTotal:       node.StakedTotal.String(),
		MaxIntendedStake:  node.MaxIntendedStake.String(),
		PendingStake:      node.PendingStake.String(),
		EpochData:         node.EpochData.EpochData,
		Status:            vega.NodeStatus(node.Status),
		Delegations:       protoDelegations,
		Name:              node.Name,
		AvatarUrl:         node.AvatarUrl,
	}

	if node.RewardScore != nil {
		res.RewardScore = node.RewardScore.ToProto()
	}

	if node.RankingScore != nil {
		res.RankingScore = node.RankingScore.ToProto()
	}

	return res
}

func (node Node) Cursor() *Cursor {
	return NewCursor(node.ID.String())
}

func (node Node) ToProtoEdge(_ ...any) (*v2.NodeEdge, error) {
	return &v2.NodeEdge{
		Node:   node.ToProto(),
		Cursor: node.Cursor().Encode(),
	}, nil
}

func (ed EpochData) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(ed)
}

func (ed *EpochData) UnmarshalJSON(b []byte) error {
	ed.EpochData = &vega.EpochData{}
	return protojson.Unmarshal(b, ed)
}

func (n *NodeData) ToProto() *vega.NodeData {
	return &vega.NodeData{
		StakedTotal:     n.StakedTotal.String(),
		TotalNodes:      n.TotalNodes,
		InactiveNodes:   n.InactiveNodes,
		ValidatingNodes: n.ValidatingNodes,
		Uptime:          float32(n.Uptime),
	}
}
