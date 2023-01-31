package spam_test

import (
	"testing"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/ptr"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	nodetypes "code.vegaprotocol.io/vega/wallet/api/node/types"
	"code.vegaprotocol.io/vega/wallet/api/spam"
	"github.com/stretchr/testify/require"
)

var (
	policies = []string{
		"proposals",
		"delegations",
		"announcements",
		"transfers",
		"votes",
	}
	proposalID = "default-proposal-id"
)

func TestSpamAwareness(t *testing.T) {
	t.Run("spam policy has been banned", testSpamPolicyBannedUtil)
	t.Run("spam policy will get banned", testSpamPolicyWillGetBanned)
	t.Run("spam policy hits limit", testSpamPolicyHitsLimit)
	t.Run("spam policy new epoch", testSpamPolicyNewEpoch)
	t.Run("spam policy pow ban", testSpamPolicyPoWBan)
	t.Run("spam policy votes separate proposals", testSpamVotesSeparateProposals)
	t.Run("other transaction types not blocked", testOtherTransactionTypesNotBlock)
}

func testSpamPolicyBannedUtil(t *testing.T) {
	p := spam.NewHandler()

	pubKey := vgcrypto.RandomHash()
	banned := nodetypes.SpamStatistic{
		BannedUntil:   ptr.From("forever"),
		MaxForEpoch:   100,
		CountForEpoch: 100,
	}

	for _, f := range policies {
		s, req := getSimplePolicyStats(t, pubKey, f, banned)
		err := p.CheckSubmission(req, s)
		require.ErrorIs(t, err, spam.ErrPartyBanned)
	}
}

func testSpamPolicyWillGetBanned(t *testing.T) {
	p := spam.NewHandler()

	pubKey := vgcrypto.RandomHash()
	willBan := nodetypes.SpamStatistic{
		BannedUntil:   nil,
		MaxForEpoch:   100,
		CountForEpoch: 100,
	}

	for _, f := range policies {
		s, req := getSimplePolicyStats(t, pubKey, f, willBan)
		err := p.CheckSubmission(req, s)
		require.ErrorIs(t, err, spam.ErrPartyWillBeBanned)
	}
}

func testSpamPolicyHitsLimit(t *testing.T) {
	p := spam.NewHandler()
	pubKey := vgcrypto.RandomHash()
	stat := nodetypes.SpamStatistic{
		BannedUntil:   nil,
		MaxForEpoch:   100,
		CountForEpoch: 99,
	}

	for _, f := range policies {
		s, req := getSimplePolicyStats(t, pubKey, f, stat)

		// we are at 99 and this check pushes to 100
		err := p.CheckSubmission(req, s)
		require.NoError(t, err)

		// so the next one will fail
		err = p.CheckSubmission(req, s)
		require.Error(t, err, spam.ErrPartyWillBeBanned)
	}
}

func testSpamPolicyNewEpoch(t *testing.T) {
	p := spam.NewHandler()
	pubKey := vgcrypto.RandomHash()
	for i, f := range policies {
		stat := nodetypes.SpamStatistic{
			BannedUntil:   nil,
			MaxForEpoch:   100,
			CountForEpoch: 88,
		}
		s, req := getSimplePolicyStats(t, pubKey, f, stat)

		// we are at 88 and this check pushes to 100
		err := p.CheckSubmission(req, s)
		require.NoError(t, err)

		// spam update is more than our count so we trust it
		stat.CountForEpoch = 100
		s, req = getSimplePolicyStats(t, pubKey, f, stat)
		err = p.CheckSubmission(req, s)
		require.Error(t, err, spam.ErrPartyWillBeBanned)

		// dip into a new epoch so we take the spam stats lower value
		stat.CountForEpoch = 0
		s, req = getSimplePolicyStats(t, pubKey, f, stat)
		s.EpochSeq += (1 + uint64(i))
		err = p.CheckSubmission(req, s)
		require.NoError(t, err)
	}
}

func testSpamPolicyPoWBan(t *testing.T) {
	p := spam.NewHandler()
	pubKey := vgcrypto.RandomHash()
	for _, f := range policies {
		stat := nodetypes.SpamStatistic{
			BannedUntil:   nil,
			MaxForEpoch:   100,
			CountForEpoch: 88,
		}
		s, req := getSimplePolicyStats(t, pubKey, f, stat)
		s.PoW.BannedUntil = ptr.From("forever")

		// we are at 88 so find to submit, but banned by PoW
		err := p.CheckSubmission(req, s)
		require.ErrorIs(t, err, spam.ErrPartyBannedPoW)
	}
}

func testSpamVotesSeparateProposals(t *testing.T) {
	p := spam.NewHandler()
	pubKey := vgcrypto.RandomHash()

	stat := nodetypes.SpamStatistic{
		BannedUntil:   nil,
		MaxForEpoch:   100,
		CountForEpoch: 100,
	}
	s, req := getSimplePolicyStats(t, pubKey, "votes", stat)

	// we're at our max for the first proposal
	err := p.CheckSubmission(req, s)
	require.ErrorIs(t, err, spam.ErrPartyWillBeBanned)

	// but can still submit against a different proposal
	req = &walletpb.SubmitTransactionRequest{
		PubKey: pubKey,
		Command: &walletpb.SubmitTransactionRequest_VoteSubmission{
			VoteSubmission: &v1.VoteSubmission{
				ProposalId: vgcrypto.RandomHash(),
			},
		},
	}
	err = p.CheckSubmission(req, s)
	require.NoError(t, err)
}

func testOtherTransactionTypesNotBlock(t *testing.T) {
	p := spam.NewHandler()
	pubKey := vgcrypto.RandomHash()
	until := ptr.From("forever")

	stat := nodetypes.SpamStatistic{
		BannedUntil: until,
	}

	// banned on everything except for proof-of-work
	stats := &nodetypes.SpamStatistics{
		PoW:               &nodetypes.PoWStatistics{},
		Proposals:         &stat,
		Delegations:       &stat,
		Transfers:         &stat,
		NodeAnnouncements: &stat,
		Votes: &nodetypes.VoteSpamStatistics{
			BannedUntil: until,
		},
	}

	// but can still submit against a different proposal
	req := &walletpb.SubmitTransactionRequest{
		PubKey:  pubKey,
		Command: &walletpb.SubmitTransactionRequest_OrderSubmission{},
	}
	err := p.CheckSubmission(req, stats)
	require.NoError(t, err)

	// but pow ban still applies
	stats.PoW.BannedUntil = until
	err = p.CheckSubmission(req, stats)
	require.ErrorIs(t, err, spam.ErrPartyBannedPoW)
}

func getSimplePolicyStats(t *testing.T, pubKey, policy string, st nodetypes.SpamStatistic) (*nodetypes.SpamStatistics, *walletpb.SubmitTransactionRequest) {
	t.Helper()
	spam := defaultSpamStats(t)
	req := &walletpb.SubmitTransactionRequest{
		PubKey: pubKey,
	}

	switch policy {
	case "proposals":
		spam.Proposals = &st
		req.Command = &walletpb.SubmitTransactionRequest_ProposalSubmission{}
	case "delegations":
		spam.Delegations = &st
		req.Command = &walletpb.SubmitTransactionRequest_DelegateSubmission{}
	case "transfers":
		spam.Transfers = &st
		req.Command = &walletpb.SubmitTransactionRequest_Transfer{}
	case "announcements":
		spam.NodeAnnouncements = &st
		req.Command = &walletpb.SubmitTransactionRequest_AnnounceNode{}
	case "votes":
		spam.Votes.MaxForEpoch = st.MaxForEpoch
		spam.Votes.BannedUntil = st.BannedUntil
		spam.Votes.Proposals = map[string]uint64{proposalID: st.CountForEpoch}
		req.Command = &walletpb.SubmitTransactionRequest_VoteSubmission{
			VoteSubmission: &v1.VoteSubmission{
				ProposalId: proposalID,
			},
		}
	default:
		t.FailNow()
	}

	return spam, req
}
