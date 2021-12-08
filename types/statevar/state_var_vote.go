package statevar

import vegapb "code.vegaprotocol.io/protos/vega"

// FloatingPointConsensusVote represents the vote of a validator on a given state variable proposal
// in response to eventID where this is the nth round of voting.
type FloatingPointConsensusVote struct {
	NodeID     string
	StateVarID string
	EventID    uint64
	Round      int64
	Vote       bool
}

// ProtoToFloatingPointConsensusVote converts to proto representation
func ProtoToFloatingPointConsensusVote(nodeID string, vote *vegapb.FloatingPointConsensusVote) *FloatingPointConsensusVote {
	return &FloatingPointConsensusVote{
		NodeID:     nodeID,
		StateVarID: vote.StateVarId,
		EventID:    vote.EventId,
		Round:      vote.Round,
		Vote:       vote.Vote,
	}
}
