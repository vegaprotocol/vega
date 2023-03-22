package spam

import (
	"errors"
	"fmt"
	"sync"

	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
)

var ErrPartyWillBeBanned = errors.New("submitting this transaction will cause this key to be temporarily banned by the the network")

type Handler struct {
	// chainID to the counter for transactions sent.
	counters map[string]*txCounter

	// chainID -> pubkey -> last known spam statistics
	spam map[string]map[string]*nodetypes.SpamStatistics

	mu sync.Mutex
}

func NewHandler() *Handler {
	return &Handler{
		counters: map[string]*txCounter{},
		spam:     map[string]map[string]*nodetypes.SpamStatistics{},
	}
}

func (s *Handler) getSpamStatisticsForChain(chainID string) map[string]*nodetypes.SpamStatistics {
	if _, ok := s.spam[chainID]; !ok {
		s.spam[chainID] = map[string]*nodetypes.SpamStatistics{}
	}
	return s.spam[chainID]
}

// checkVote because it has to be a little different...
func (s *Handler) checkVote(propID string, st *nodetypes.VoteSpamStatistics) error {
	if st.BannedUntil != nil {
		return fmt.Errorf("party is banned from submitting transactions of this type until %s", *st.BannedUntil)
	}
	v := st.Proposals[propID]
	if v == st.MaxForEpoch {
		return fmt.Errorf("party has already submitted the maximum number of transactions of this type per epoch (%d)", st.MaxForEpoch)
	}
	st.Proposals[propID]++
	return nil
}

func (s *Handler) checkTxn(st *nodetypes.SpamStatistic) error {
	if st.BannedUntil != nil {
		return fmt.Errorf("party is banned from submitting transactions of this type until %s", *st.BannedUntil)
	}

	if st.CountForEpoch == st.MaxForEpoch {
		return fmt.Errorf("party has already submitted the maximum number of transactions of this type per epoch (%d)", st.MaxForEpoch)
	}

	// increment the count by hand because the spam-stats endpoint only updates once a block
	// so if we send in multiple transactions between that next update we need to know about
	// the past ones
	st.CountForEpoch++
	return nil
}

func (s *Handler) mergeVotes(st *nodetypes.VoteSpamStatistics, other *nodetypes.VoteSpamStatistics) {
	st.BannedUntil = other.BannedUntil
	st.MaxForEpoch = other.MaxForEpoch
	for pid, cnt := range other.Proposals {
		if cnt > st.Proposals[pid] {
			st.Proposals[pid] = cnt
		}
	}
}

// merge will take the spam stats from other and update st only if other's counters are higher.
func (s *Handler) merge(st *nodetypes.SpamStatistic, other *nodetypes.SpamStatistic) {
	st.BannedUntil = other.BannedUntil
	st.MaxForEpoch = other.MaxForEpoch

	// we've pinged the spam endpoint and the count it returns will either
	// 1) equal our counts and we're fine
	// 2) have a bigger count then ours, meaning something external has submitted for, so we take the bigger count
	// 3) its count is smaller than ours meaning we're submitting lots on the same block and so the spam endpoint is behind,
	//    so we keep what we have
	if other.CountForEpoch > st.CountForEpoch {
		st.CountForEpoch = other.CountForEpoch
	}
}

// CheckSubmission return an error if we are banned from making this type of transaction or if submitting
// the transaction will result in a banning.
func (s *Handler) CheckSubmission(req *walletpb.SubmitTransactionRequest, newStats *nodetypes.SpamStatistics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	chainStats := s.getSpamStatisticsForChain(newStats.ChainID)

	stats, ok := chainStats[req.PubKey]
	if !ok {
		chainStats[req.PubKey] = newStats
		stats = newStats
	}

	if stats.EpochSeq < newStats.EpochSeq {
		// we can reset all the spam statistics now that we're in a new epoch and just take what the spam endpoint tells us
		chainStats[req.PubKey] = newStats
		stats = newStats
	}

	if newStats.PoW.BannedUntil != nil {
		return fmt.Errorf("party is banned from submitting all transactions until %s", *newStats.PoW.BannedUntil)
	}

	switch cmd := req.Command.(type) {
	case *walletpb.SubmitTransactionRequest_ProposalSubmission:
		s.merge(stats.Proposals, newStats.Proposals)
		return s.checkTxn(stats.Proposals)
	case *walletpb.SubmitTransactionRequest_AnnounceNode:
		s.merge(stats.NodeAnnouncements, newStats.NodeAnnouncements)
		return s.checkTxn(stats.NodeAnnouncements)
	case *walletpb.SubmitTransactionRequest_UndelegateSubmission, *walletpb.SubmitTransactionRequest_DelegateSubmission:
		s.merge(stats.Delegations, newStats.Delegations)
		return s.checkTxn(stats.Delegations)
	case *walletpb.SubmitTransactionRequest_Transfer:
		s.merge(stats.Transfers, newStats.Transfers)
		return s.checkTxn(stats.Transfers)
	case *walletpb.SubmitTransactionRequest_IssueSignatures:
		s.merge(stats.IssuesSignatures, newStats.IssuesSignatures)
		return s.checkTxn(stats.IssuesSignatures)
	case *walletpb.SubmitTransactionRequest_VoteSubmission:
		s.mergeVotes(stats.Votes, newStats.Votes)
		return s.checkVote(cmd.VoteSubmission.ProposalId, stats.Votes)
	}

	return nil
}
