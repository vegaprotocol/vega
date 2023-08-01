package snapshot_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	typemocks "code.vegaprotocol.io/vega/core/types/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/cosmos/iavl"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type snapshotForTest struct {
	snapshot   *types.Snapshot
	appState   *types.AppState
	byteChunks [][]byte
	rawChunks  []*types.RawChunk
	payloads   []*types.Payload
}

func (s *snapshotForTest) PayloadGovernanceActive() *types.Payload {
	for _, payload := range s.payloads {
		switch payload.Data.(type) {
		case *types.PayloadGovernanceActive:
			return payload
		}
	}

	return nil
}

func (s *snapshotForTest) PayloadGovernanceEnacted() *types.Payload {
	for _, payload := range s.payloads {
		switch payload.Data.(type) {
		case *types.PayloadGovernanceEnacted:
			return payload
		}
	}

	return nil
}

func (s *snapshotForTest) PayloadDelegationActive() *types.Payload {
	for _, payload := range s.payloads {
		switch payload.Data.(type) {
		case *types.PayloadDelegationActive:
			return payload
		}
	}

	return nil
}

func (s *snapshotForTest) PayloadEpoch() *types.Payload {
	for _, payload := range s.payloads {
		switch payload.Data.(type) {
		case *types.PayloadEpoch:
			return payload
		}
	}

	return nil
}

func newEpochProvider(t *testing.T, ctrl *gomock.Controller) *typemocks.MockStateProvider {
	t.Helper()

	epochProvider := typemocks.NewMockStateProvider(ctrl)
	epochProvider.EXPECT().Namespace().Return(types.SnapshotNamespace("epoch")).AnyTimes()
	epochProvider.EXPECT().Keys().Return([]string{"all"}).AnyTimes()
	epochProvider.EXPECT().Stopped().Return(false).AnyTimes()
	return epochProvider
}

func newDelegationProvider(t *testing.T, ctrl *gomock.Controller) *typemocks.MockStateProvider {
	t.Helper()

	delegationProvider := typemocks.NewMockStateProvider(ctrl)
	delegationProvider.EXPECT().Namespace().Return(types.SnapshotNamespace("delegation")).AnyTimes()
	delegationProvider.EXPECT().Keys().Return([]string{"active"}).AnyTimes()
	delegationProvider.EXPECT().Stopped().Return(false).AnyTimes()
	return delegationProvider
}

func newGovernanceProvider(t *testing.T, ctrl *gomock.Controller) *typemocks.MockStateProvider {
	t.Helper()

	governanceProvider := typemocks.NewMockStateProvider(ctrl)
	governanceProvider.EXPECT().Namespace().Return(types.SnapshotNamespace("governance")).AnyTimes()
	governanceProvider.EXPECT().Keys().Return([]string{"active", "enacted"}).AnyTimes()
	governanceProvider.EXPECT().Stopped().Return(false).AnyTimes()
	return governanceProvider
}

func firstSnapshot(t *testing.T) *snapshotForTest {
	t.Helper()

	appState, payloads := firstState(t)

	return toSnapshotTest(t, payloads, appState)
}

func secondSnapshot(t *testing.T) *snapshotForTest {
	t.Helper()

	appState, payloads := secondState(t)

	return toSnapshotTest(t, payloads, appState)
}

func toSnapshotTest(t *testing.T, payloads []*types.Payload, appState *types.AppState) *snapshotForTest {
	t.Helper()

	tree, err := iavl.NewMutableTree(cometbftdb.NewMemDB(), 0, false)
	require.NoError(t, err)

	_, err = tree.Load()
	require.NoError(t, err)

	for _, p := range payloads {
		payload, err := proto.Marshal(p.IntoProto())
		require.NoError(t, err)
		_, err = tree.Set([]byte(p.TreeKey()), payload)
		require.NoError(t, err)
	}

	_, _, err = tree.SaveVersion()
	require.NoError(t, err)

	s, err := types.SnapshotFromTree(tree.ImmutableTree)
	require.NoError(t, err)

	rawChunks := make([]*types.RawChunk, 0, s.Chunks)
	for i := uint32(0); i < s.Chunks; i++ {
		rawChunks = append(rawChunks, &types.RawChunk{
			Nr:     i,
			Data:   s.ByteChunks[int(i)],
			Height: s.Height,
			Format: s.Format,
		})
	}

	return &snapshotForTest{
		snapshot: &types.Snapshot{
			Format: s.Format,
			Height: s.Height,
			Hash:   s.Hash,
			Meta: &types.Metadata{
				Version:     s.Meta.Version,
				NodeHashes:  s.Meta.NodeHashes,
				ChunkHashes: s.Meta.ChunkHashes,
			},
			Chunks: s.Chunks,
		},
		byteChunks: s.ByteChunks,
		rawChunks:  rawChunks,
		payloads:   payloads,
		appState:   appState,
	}
}

func firstState(t *testing.T) (*types.AppState, []*types.Payload) {
	t.Helper()

	chainTime, err := time.Parse("2006-01-02 15:04", "2022-12-12 04:35")
	require.NoError(t, err)

	appState := &types.AppState{
		Height:  64,
		Block:   "qwertyuiop1234567890",
		Time:    chainTime.UnixNano(),
		ChainID: "testnet-1",
	}

	// Random data for active governance.
	activeProposalID := vgrand.RandomStr(5)
	activeProposalTime, err := time.Parse("2006-01-02 15:04", "2022-12-15 06:30")
	require.NoError(t, err)

	// Random data for enacted governance.
	enactedProposalID := vgrand.RandomStr(5)
	enactedProposalTime, err := time.Parse("2006-01-02 15:04", "2022-12-10 06:30")
	require.NoError(t, err)

	payloads := []*types.Payload{
		{
			Data: &types.PayloadAppState{
				AppState: appState,
			},
		},
		{
			Data: &types.PayloadGovernanceActive{
				GovernanceActive: &types.GovernanceActive{
					Proposals: []*types.ProposalData{
						{
							Proposal: &types.Proposal{
								ID:        activeProposalID,
								Reference: vgrand.RandomStr(5),
								Party:     vgrand.RandomStr(5),
								State:     types.ProposalStatePassed,
								Timestamp: activeProposalTime.UnixNano(),
								Terms: &types.ProposalTerms{
									ClosingTimestamp:    activeProposalTime.Add(1 * time.Hour).UnixNano(),
									EnactmentTimestamp:  activeProposalTime.Add(2 * time.Hour).UnixNano(),
									ValidationTimestamp: 0,
									Change: &types.ProposalTermsUpdateNetworkParameter{
										UpdateNetworkParameter: &types.UpdateNetworkParameter{
											Changes: &types.NetworkParameter{
												Key:   vgrand.RandomStr(10),
												Value: vgrand.RandomStr(10),
											},
										},
									},
								},
								Rationale: &types.ProposalRationale{
									Description: vgrand.RandomStr(10),
									Title:       vgrand.RandomStr(10),
								},
								Reason:                  0,
								ErrorDetails:            "",
								RequiredMajority:        num.DecimalFromInt64(100),
								RequiredParticipation:   num.DecimalFromInt64(50),
								RequiredLPMajority:      num.DecimalFromInt64(100),
								RequiredLPParticipation: num.DecimalFromInt64(50),
							},
							Yes: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(31 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							No: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(32 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(33 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(34 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							Invalid: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(35 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(36 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
						},
					},
				},
			},
		},
		{
			Data: &types.PayloadGovernanceEnacted{
				GovernanceEnacted: &types.GovernanceEnacted{
					Proposals: []*types.ProposalData{
						{
							Proposal: &types.Proposal{
								ID:        enactedProposalID,
								Reference: vgrand.RandomStr(5),
								Party:     vgrand.RandomStr(5),
								State:     types.ProposalStatePassed,
								Timestamp: enactedProposalTime.UnixNano(),
								Terms: &types.ProposalTerms{
									ClosingTimestamp:    enactedProposalTime.Add(1 * time.Hour).UnixNano(),
									EnactmentTimestamp:  enactedProposalTime.Add(2 * time.Hour).UnixNano(),
									ValidationTimestamp: 0,
									Change: &types.ProposalTermsUpdateNetworkParameter{
										UpdateNetworkParameter: &types.UpdateNetworkParameter{
											Changes: &types.NetworkParameter{
												Key:   vgrand.RandomStr(10),
												Value: vgrand.RandomStr(10),
											},
										},
									},
								},
								Rationale: &types.ProposalRationale{
									Description: vgrand.RandomStr(10),
									Title:       vgrand.RandomStr(10),
								},
								Reason:                  0,
								ErrorDetails:            "",
								RequiredMajority:        num.DecimalFromInt64(100),
								RequiredParticipation:   num.DecimalFromInt64(50),
								RequiredLPMajority:      num.DecimalFromInt64(100),
								RequiredLPParticipation: num.DecimalFromInt64(50),
							},
							Yes: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   enactedProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   enactedProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   enactedProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   enactedProposalTime.Add(31 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							No: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   enactedProposalTime.Add(32 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   enactedProposalTime.Add(33 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   enactedProposalTime.Add(34 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							Invalid: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   enactedProposalTime.Add(35 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  enactedProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   enactedProposalTime.Add(36 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
						},
					},
				},
			},
		},
		{
			Data: &types.PayloadDelegationActive{
				DelegationActive: &types.DelegationActive{
					Delegations: []*types.Delegation{
						{
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("1000"),
							EpochSeq: "10",
						}, {
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("2000"),
							EpochSeq: "20",
						}, {
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("3000"),
							EpochSeq: "30",
						},
					},
				},
			},
		},
		{
			Data: &types.PayloadEpoch{
				EpochState: &types.EpochState{
					Seq:                  7,
					StartTime:            time.Now().In(time.UTC),
					ExpireTime:           time.Now().Add(1 * time.Hour).In(time.UTC),
					ReadyToStartNewEpoch: true,
					ReadyToEndEpoch:      false,
				},
			},
		},
	}
	return appState, payloads
}

func secondState(t *testing.T) (*types.AppState, []*types.Payload) {
	t.Helper()

	chainTime, err := time.Parse("2006-01-02 15:04", "2022-12-24 04:35")
	require.NoError(t, err)

	appState := &types.AppState{
		Height:  164,
		Block:   "1234567890qwertyuiop",
		Time:    chainTime.UnixNano(),
		ChainID: "testnet-1",
	}

	// Random data for active governance.
	activeProposalID := vgrand.RandomStr(5)
	activeProposalTime, err := time.Parse("2006-01-02 15:04", "2022-12-25 06:30")
	require.NoError(t, err)

	payloads := []*types.Payload{
		{
			Data: &types.PayloadAppState{
				AppState: appState,
			},
		},
		{
			Data: &types.PayloadGovernanceActive{
				GovernanceActive: &types.GovernanceActive{
					Proposals: []*types.ProposalData{
						{
							Proposal: &types.Proposal{
								ID:        activeProposalID,
								Reference: vgrand.RandomStr(5),
								Party:     vgrand.RandomStr(5),
								State:     types.ProposalStatePassed,
								Timestamp: activeProposalTime.UnixNano(),
								Terms: &types.ProposalTerms{
									ClosingTimestamp:    activeProposalTime.Add(1 * time.Hour).UnixNano(),
									EnactmentTimestamp:  activeProposalTime.Add(2 * time.Hour).UnixNano(),
									ValidationTimestamp: 0,
									Change: &types.ProposalTermsUpdateNetworkParameter{
										UpdateNetworkParameter: &types.UpdateNetworkParameter{
											Changes: &types.NetworkParameter{
												Key:   vgrand.RandomStr(10),
												Value: vgrand.RandomStr(10),
											},
										},
									},
								},
								Rationale: &types.ProposalRationale{
									Description: vgrand.RandomStr(10),
									Title:       vgrand.RandomStr(10),
								},
								Reason:                  0,
								ErrorDetails:            "",
								RequiredMajority:        num.DecimalFromInt64(100),
								RequiredParticipation:   num.DecimalFromInt64(50),
								RequiredLPMajority:      num.DecimalFromInt64(100),
								RequiredLPParticipation: num.DecimalFromInt64(50),
							},
							Yes: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(30 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(31 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							No: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(32 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(33 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(34 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
							Invalid: []*types.Vote{
								{
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueYes,
									Timestamp:                   activeProposalTime.Add(35 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								}, {
									PartyID:                     vgrand.RandomStr(5),
									ProposalID:                  activeProposalID,
									Value:                       types.VoteValueNo,
									Timestamp:                   activeProposalTime.Add(36 * time.Minute).UnixNano(),
									TotalGovernanceTokenBalance: toUint("100"),
									TotalGovernanceTokenWeight:  num.DecimalFromInt64(100),
									TotalEquityLikeShareWeight:  num.DecimalFromInt64(100),
								},
							},
						},
					},
				},
			},
		},
		{
			// Nothing set on purpose, to see if the snapshot behaves properly
			// when a key becomes empty.
			Data: &types.PayloadGovernanceEnacted{
				GovernanceEnacted: &types.GovernanceEnacted{},
			},
		},
		{
			Data: &types.PayloadDelegationActive{
				DelegationActive: &types.DelegationActive{
					Delegations: []*types.Delegation{
						{
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("1000"),
							EpochSeq: "10",
						}, {
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("2000"),
							EpochSeq: "20",
						}, {
							Party:    vgrand.RandomStr(5),
							NodeID:   vgrand.RandomStr(5),
							Amount:   toUint("3000"),
							EpochSeq: "30",
						},
					},
				},
			},
		},
		{
			Data: &types.PayloadEpoch{
				EpochState: &types.EpochState{
					Seq:                  7,
					StartTime:            time.Now().In(time.UTC),
					ExpireTime:           time.Now().Add(1 * time.Hour).In(time.UTC),
					ReadyToStartNewEpoch: true,
					ReadyToEndEpoch:      false,
				},
			},
		},
	}
	return appState, payloads
}

func toUint(n string) *num.Uint {
	uintNum, _ := num.UintFromString(n, 10)
	return uintNum
}

func serialize(t *testing.T, payload *types.Payload) []byte {
	t.Helper()

	serializedPayload, err := proto.Marshal(payload.IntoProto())
	require.NoError(t, err)
	return serializedPayload
}
