package processor_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/require"
)

func TestUpdateMaxGas(t *testing.T) {
	eet := &ExecEngineTest{marketCounters: map[string]*types.MarketCounters{}}
	gastimator := processor.NewGastimator(eet)
	gastimator.OnMaxGasUpdate(context.Background(), num.NewUint(1234))
	require.Equal(t, uint64(1234), gastimator.GetMaxGas())
}

func TestSubmitOrder(t *testing.T) {
	tx := &testTx{
		command:      txn.SubmitOrderCommand,
		unmarshaller: unmarshalSubmitOrder(&commandspb.OrderSubmission{MarketId: "1"}),
	}
	testOrderSubmitAmendAndLP(t, tx)
}

func TestAmendOrder(t *testing.T) {
	tx := &testTx{
		command:      txn.AmendOrderCommand,
		unmarshaller: unmarshalAmendtOrder(&commandspb.OrderAmendment{MarketId: "1"}),
	}
	testOrderSubmitAmendAndLP(t, tx)
}

func TestSubmitLP(t *testing.T) {
	tx := &testTx{
		command:      txn.LiquidityProvisionCommand,
		unmarshaller: unmarshalLPSubmission(&commandspb.LiquidityProvisionSubmission{MarketId: "1"}),
	}
	testOrderSubmitAmendAndLP(t, tx)
}

func TestAmendLP(t *testing.T) {
	tx := &testTx{
		command:      txn.AmendLiquidityProvisionCommand,
		unmarshaller: unmarshalLPAmendment(&commandspb.LiquidityProvisionAmendment{MarketId: "1"}),
	}
	testOrderSubmitAmendAndLP(t, tx)
}

func TestCancelLP(t *testing.T) {
	tx := &testTx{
		command:      txn.CancelLiquidityProvisionCommand,
		unmarshaller: unmarshalLPCancellation(&commandspb.LiquidityProvisionCancellation{MarketId: "1"}),
	}
	testOrderSubmitAmendAndLP(t, tx)
}

func testOrderSubmitAmendAndLP(t *testing.T, tx *testTx) {
	t.Helper()
	marketCounters := map[string]*types.MarketCounters{}
	eet := &ExecEngineTest{marketCounters: marketCounters}
	gastimator := processor.NewGastimator(eet)
	gastimator.OnMaxGasUpdate(context.Background(), num.NewUint(1234))
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(1))
	gastimator.OnMinBlockCapacityUpdate(context.Background(), num.NewUint(1))

	// there's nothing yet for the market so expect default counters
	count, err := gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// change the default gas to see we get the new default
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(10))
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(10), count)

	// set some counters
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  1,
		PositionCount:       2,
		OrderbookLevelCount: 10,
	}
	gastimator.OnBlockEnd()

	// gasOrder = network.transaction.defaultgas + peg cost factor x pegs
	//                                         + LP shape cost factor x shapes
	//                                         + position factor x positions
	//                                         + level factor x levels
	// gasOrder = min(maxGas-1,gasOrder)
	// gasOrder = min(1233, 10 + 50 * 1 + 100 * 5 + 2 + 10 * 0.1) = 563
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(563), count)

	// update counters such that now the max gas is lower than gas wanted for the order
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  8,
		PositionCount:       2,
		OrderbookLevelCount: 100,
	}

	// gasOrder = min(1233, 10 + 50 * 8 + 100 * 10 + 2 + 100 * 0.1) = 1233
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(1233), count)
}

func TestCancelOrder(t *testing.T) {
	tx := &testTx{
		command:      txn.CancelOrderCommand,
		unmarshaller: unmarshalCancelOrder(&commandspb.OrderCancellation{MarketId: "1", OrderId: "1"}),
	}

	marketCounters := map[string]*types.MarketCounters{}
	eet := &ExecEngineTest{marketCounters: marketCounters}
	gastimator := processor.NewGastimator(eet)
	gastimator.OnMaxGasUpdate(context.Background(), num.NewUint(1234))
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(1))
	gastimator.OnMinBlockCapacityUpdate(context.Background(), num.NewUint(1))

	// there's nothing yet for the market so expect default counters
	count, err := gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// change the default gas to see we get the new default
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(10))
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(10), count)

	// set some counters
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  1,
		PositionCount:       2,
		OrderbookLevelCount: 10,
	}
	gastimator.OnBlockEnd()

	// gasCancel = network.transaction.defaultgas + peg cost factor x pegs
	// 	+ LP shape cost factor x shapes
	// 	+ level factor x levels
	// gasCancel = min(maxGas-1,gasCancel)
	// gasOrder = min(1233, 10 + 50 * 1 + 100 * 5 + 10 * 0.1) = 561
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(561), count)

	// update counters such that now the max gas is lower than gasCancel
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  8,
		PositionCount:       2,
		OrderbookLevelCount: 100,
	}

	// gasOrder = min(1233, 10 + 50 * 8 + 100 * 10 + 100 * 0.1) = 1233
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(1233), count)
}

func TestBatch(t *testing.T) {
	tx := &testTx{
		command: txn.BatchMarketInstructions,
		unmarshaller: unmarshalBatch(&commandspb.BatchMarketInstructions{
			Submissions:   []*commandspb.OrderSubmission{{MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}},
			Cancellations: []*commandspb.OrderCancellation{{MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}},
			Amendments:    []*commandspb.OrderAmendment{{MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}, {MarketId: "1"}},
		},
		),
	}

	marketCounters := map[string]*types.MarketCounters{}
	eet := &ExecEngineTest{marketCounters: marketCounters}
	gastimator := processor.NewGastimator(eet)
	gastimator.OnMaxGasUpdate(context.Background(), num.NewUint(10000))
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(1))
	gastimator.OnMinBlockCapacityUpdate(context.Background(), num.NewUint(1))

	// there's nothing yet for any market so expect defaultgas * 3 + 4 * defaultgas = 7 * defaultgas
	count, err := gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(7), count)

	// change the default gas to see we get the new default
	// defaultGas + 0.5 * 2 * defaultGas +
	// defaultGas + 2 * defaultGas +
	// defaultGas + 1.5 * defaultGas = 75
	gastimator.OnDefaultGasUpdate(context.Background(), num.NewUint(10))
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(75), count)

	// set some counters
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  1,
		PositionCount:       2,
		OrderbookLevelCount: 10,
	}

	gastimator.OnBlockEnd()

	// we have 3 submissions, 5 cancellations and 4 amendments
	// gas = 563*2 + 561 * 3 + 563 * 2.5 = 4216
	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(4216), count)

	// update counters such that now the max gas is lower than gasCancel
	marketCounters["1"] = &types.MarketCounters{
		PeggedOrderCounter:  8,
		PositionCount:       2,
		OrderbookLevelCount: 100,
	}

	count, err = gastimator.CalcGasWantedForTx(tx)
	require.NoError(t, err)
	require.Equal(t, uint64(9999), count)
}

func TestGetPriority(t *testing.T) {
	command := []txn.Command{
		txn.SubmitOrderCommand,
		txn.CancelOrderCommand,
		txn.AmendOrderCommand,
		txn.WithdrawCommand,
		txn.ProposeCommand,
		txn.VoteCommand,
		txn.AnnounceNodeCommand,
		txn.NodeVoteCommand,
		txn.NodeSignatureCommand,
		txn.LiquidityProvisionCommand,
		txn.CancelLiquidityProvisionCommand,
		txn.AmendLiquidityProvisionCommand,
		txn.ChainEventCommand,
		txn.SubmitOracleDataCommand,
		txn.DelegateCommand,
		txn.UndelegateCommand,
		txn.RotateKeySubmissionCommand,
		txn.StateVariableProposalCommand,
		txn.TransferFundsCommand,
		txn.CancelTransferFundsCommand,
		txn.ValidatorHeartbeatCommand,
		txn.RotateEthereumKeySubmissionCommand,
		txn.ProtocolUpgradeCommand,
		txn.IssueSignatures,
		txn.BatchMarketInstructions,
	}
	marketCounters := map[string]*types.MarketCounters{}
	eet := &ExecEngineTest{marketCounters: marketCounters}
	gastimator := processor.NewGastimator(eet)
	for _, c := range command {
		expected := uint64(1)
		if c.IsValidatorCommand() {
			expected = uint64(10000)
		} else if c == txn.ProposeCommand || c == txn.VoteCommand {
			expected = uint64(100)
		}
		require.Equal(t, expected, gastimator.GetPriority(&testTx{command: c}), c)
	}
}

type ExecEngineTest struct {
	marketCounters map[string]*types.MarketCounters
}

func (eet *ExecEngineTest) GetMarketCounters() map[string]*types.MarketCounters {
	return eet.marketCounters
}

func unmarshalBatch(batch *commandspb.BatchMarketInstructions) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.BatchMarketInstructions)
		*underlyingCmd = *batch
		return nil
	}
}

func unmarshalSubmitOrder(order *commandspb.OrderSubmission) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.OrderSubmission)
		*underlyingCmd = *order
		return nil
	}
}

func unmarshalAmendtOrder(order *commandspb.OrderAmendment) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.OrderAmendment)
		*underlyingCmd = *order
		return nil
	}
}

func unmarshalCancelOrder(order *commandspb.OrderCancellation) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.OrderCancellation)
		*underlyingCmd = *order
		return nil
	}
}

func unmarshalLPSubmission(order *commandspb.LiquidityProvisionSubmission) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.LiquidityProvisionSubmission)
		*underlyingCmd = *order
		return nil
	}
}

func unmarshalLPAmendment(order *commandspb.LiquidityProvisionAmendment) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.LiquidityProvisionAmendment)
		*underlyingCmd = *order
		return nil
	}
}

func unmarshalLPCancellation(order *commandspb.LiquidityProvisionCancellation) func(interface{}) error {
	return func(i interface{}) error {
		underlyingCmd, _ := i.(*commandspb.LiquidityProvisionCancellation)
		*underlyingCmd = *order
		return nil
	}
}

type testTx struct {
	command      txn.Command
	unmarshaller func(interface{}) error
}

func (tx *testTx) Unmarshal(i interface{}) error { return tx.unmarshaller(i) }
func (tx *testTx) GetPoWTID() string             { return "" }
func (tx *testTx) GetVersion() uint32            { return 2 }
func (tx *testTx) GetPoWNonce() uint64           { return 0 }
func (tx *testTx) Signature() []byte             { return []byte{} }
func (tx *testTx) Payload() []byte               { return nil }
func (tx *testTx) PubKey() []byte                { return []byte{} }
func (tx *testTx) PubKeyHex() string             { return "" }
func (tx *testTx) Party() string                 { return "" }
func (tx *testTx) Hash() []byte                  { return []byte{} }
func (tx *testTx) Command() txn.Command          { return tx.command }
func (tx *testTx) BlockHeight() uint64           { return 0 }
func (tx *testTx) GetCmd() interface{}           { return nil }
