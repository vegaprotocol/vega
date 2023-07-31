package protocolupgrade_test

import (
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/protocolupgrade"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/libs/num"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestValidatorToplogy struct {
	totalVotingPower int64
}

func (vt *TestValidatorToplogy) IsSelfTendermintValidator() bool          { return true }
func (vt *TestValidatorToplogy) IsTendermintValidator(pubkey string) bool { return true }
func (vt *TestValidatorToplogy) GetVotingPower(pubkey string) int64       { return 10 }
func (vt *TestValidatorToplogy) GetTotalVotingPower() int64               { return vt.totalVotingPower }

func testEngine(t *testing.T, vegaPath paths.Paths) (*protocolupgrade.Engine, *snp.Engine, *bmocks.MockBroker, *TestValidatorToplogy) {
	t.Helper()
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	now := time.Now()
	log := logging.NewTestLogger()
	testTopology := &TestValidatorToplogy{}
	engine := protocolupgrade.New(log, protocolupgrade.NewDefaultConfig(), broker, testTopology, "0.54.0")
	engine.OnRequiredMajorityChanged(context.Background(), num.DecimalFromFloat(0.66))
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	snapshotEngine, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine.AddProviders(engine)
	return engine, snapshotEngine, broker, testTopology
}

func Test(t *testing.T) {
	t.Run("Upgrade proposal gets rejected", testUpgradeProposalRejected)
	t.Run("Upgrade proposal gets accepted", testProposalApproved)
	t.Run("Multiple upgrade proposal get accepted, earliest is chosen", testMultiProposalApproved)
	t.Run("Snapshot roundtrip test", testSnapshotRoundTrip)
	t.Run("Revert a proposal", testRevertProposal)
	t.Run("Downgrade is not allowed", testDowngradeVersionNotAllowed)
}

func testDowngradeVersionNotAllowed(t *testing.T) {
	e, _, broker, _ := testEngine(t, paths.New(t.TempDir()))
	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()
	// validator1 proposed an upgrade to v1 at block height 100
	require.EqualError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "0.53.0"), "upgrade version is too old")
}

func testRevertProposal(t *testing.T) {
	e, _, broker, _ := testEngine(t, paths.New(t.TempDir()))
	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()
	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "1.0.0"))
	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[0].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 1, len(evts[0].StreamMessage().GetProtocolUpgradeEvent().Approvers))

	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "0.54.0"))

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED, evts[1].StreamMessage().GetProtocolUpgradeEvent().Status)

	require.Equal(t, 0, len(evts[1].StreamMessage().GetProtocolUpgradeEvent().Approvers))
}

func testUpgradeProposalRejected(t *testing.T) {
	e, _, broker, testTopology := testEngine(t, paths.New(t.TempDir()))
	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()

	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "1.0.0"))
	// validator2 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk2", 100, "1.0.0"))
	// validator3 proposed an upgrade to v2 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk3", 100, "1.0.2"))

	// we reached block 100 and only 50% (<66%) of the voting power agreed so the proposal is rejected
	testTopology.totalVotingPower = 40
	e.BeginBlock(context.Background(), 100)
	require.Equal(t, 5, len(evts))

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[0].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 1, len(evts[0].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.0", evts[0].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[1].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 2, len(evts[1].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.0", evts[1].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[2].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 1, len(evts[2].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.2", evts[2].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED, evts[3].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, "1.0.0", evts[3].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED, evts[4].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, "1.0.2", evts[4].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.False(t, e.TimeForUpgrade())
}

func testProposalApproved(t *testing.T) {
	e, _, broker, testTopology := testEngine(t, paths.New(t.TempDir()))
	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()

	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "1.0.0"))
	// validator2 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk2", 100, "1.0.0"))
	// validator3 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk3", 100, "1.0.0"))

	// full house
	testTopology.totalVotingPower = 30

	e.BeginBlock(context.Background(), 50)
	require.Equal(t, 3, len(evts))

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[0].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 1, len(evts[0].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.0", evts[0].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[1].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 2, len(evts[1].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.0", evts[1].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING, evts[2].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, 3, len(evts[2].StreamMessage().GetProtocolUpgradeEvent().Approvers))
	require.Equal(t, "1.0.0", evts[2].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	e.BeginBlock(context.Background(), 100)
	require.True(t, e.TimeForUpgrade())
	e.Cleanup(context.Background())
	require.Equal(t, 4, len(evts))
	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED, evts[3].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, "1.0.0", evts[3].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)

	e.SetCoreReadyForUpgrade()
	e.SetReadyForUpgrade()
}

func testMultiProposalApproved(t *testing.T) {
	e, _, broker, testTopology := testEngine(t, paths.New(t.TempDir()))
	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()

	testTopology.totalVotingPower = 20

	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 100, "1.0.0"))
	// validator2 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk2", 100, "1.0.0"))
	// validator3 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk3", 100, "1.0.0"))

	require.Equal(t, 3, len(evts[2].StreamMessage().GetProtocolUpgradeEvent().Approvers))

	// validator1 also proposed an upgrade to v1 at block height 90
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk1", 90, "1.0.1"))

	// the new proposal from pk1 voids their approval of the former proposal
	require.Equal(t, 2, len(evts[4].StreamMessage().GetProtocolUpgradeEvent().Approvers))

	// validator2 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk2", 90, "1.0.1"))

	// the new proposal from pk1 voids their approval of the former proposal
	require.Equal(t, 1, len(evts[6].StreamMessage().GetProtocolUpgradeEvent().Approvers))

	// validator3 agrees
	require.NoError(t, e.UpgradeProposal(context.Background(), "pk3", 90, "1.0.1"))

	// at this point there are no votes for the proposal for 1.0.0 so it gets removed

	// the new proposal from pk1 voids their approval of the former proposal
	require.Equal(t, 0, len(evts[8].StreamMessage().GetProtocolUpgradeEvent().Approvers))

	e.BeginBlock(context.Background(), 55)
	require.Equal(t, 9, len(evts))

	e.BeginBlock(context.Background(), 90)
	e.Cleanup(context.Background())
	require.Equal(t, 10, len(evts))
	require.True(t, e.TimeForUpgrade())

	require.Equal(t, eventspb.ProtocolUpgradeProposalStatus_PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED, evts[9].StreamMessage().GetProtocolUpgradeEvent().Status)
	require.Equal(t, "1.0.1", evts[9].StreamMessage().GetProtocolUpgradeEvent().VegaReleaseTag)
	require.Equal(t, uint64(90), evts[9].StreamMessage().GetProtocolUpgradeEvent().UpgradeBlockHeight)
}

func testSnapshotRoundTrip(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	vegaPath := paths.New(t.TempDir())
	puEngine1, snapshotEngine1, broker, _ := testEngine(t, vegaPath)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()
	require.NoError(t, snapshotEngine1.Start(ctx))

	var evts []events.Event
	broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(event events.Event) {
		evts = append(evts, event)
	}).AnyTimes()

	puEngine1.BeginBlock(ctx, 50)

	// validator1 proposed an upgrade to v1 at block height 100
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk1", 100, "1.0.0"))
	// validator2 agrees
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk2", 100, "1.0.0"))
	// validator3 agrees
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk3", 100, "1.0.0"))

	// validator1 also proposed an upgrade to v1 at block height 90
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk1", 90, "1.0.1"))
	// validator2 agrees
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk2", 90, "1.0.1"))
	// validator3 agrees
	require.NoError(t, puEngine1.UpgradeProposal(context.Background(), "pk3", 90, "1.0.1"))

	// take a snapshot
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	puEngine1.BeginBlock(context.Background(), 91)

	state1 := map[string][]byte{}
	for _, key := range puEngine1.Keys() {
		state, additionalProvider, err := puEngine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	puEngine2, snapshotEngine2, brokerLoad, _ := testEngine(t, vegaPath)
	brokerLoad.EXPECT().Send(gomock.Any()).AnyTimes()

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	puEngine2.BeginBlock(context.Background(), 91)

	state2 := map[string][]byte{}
	for _, key := range puEngine2.Keys() {
		state, additionalProvider, err := puEngine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
