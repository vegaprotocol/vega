package nodewallets

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/commands"
	api "code.vegaprotocol.io/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets/vega"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
)

const (
	commanderNamedLogger = "commander"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/nodewallets Chain
type Chain interface {
	SubmitTransaction(ctx context.Context, tx *commandspb.Transaction, ty api.SubmitTransactionRequest_Type) (string, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_stats_mock.go -package mocks code.vegaprotocol.io/vega/nodewallets BlockchainStats
type BlockchainStats interface {
	Height() uint64
}

type Commander struct {
	log    *logging.Logger
	bc     Chain
	wallet *vega.Wallet
	bstats BlockchainStats
}

// NewCommander - used to sign and send transaction from core
// e.g. NodeRegistration, NodeVote
// chain argument can't be passed in cmd package, but is used for tests.
func NewCommander(cfg Config, log *logging.Logger, bc Chain, w *vega.Wallet, bstats BlockchainStats) (*Commander, error) {
	log = log.Named(commanderNamedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Commander{
		log:    log,
		bc:     bc,
		wallet: w,
		bstats: bstats,
	}, nil
}

// SetChain - currently need to hack around the chicken/egg problem.
func (c *Commander) SetChain(bc *blockchain.Client) {
	c.bc = bc
}

// Command - send command to chain.
func (c *Commander) Command(ctx context.Context, cmd txn.Command, payload proto.Message, done func(error)) {
	c.command(ctx, cmd, payload, done, api.SubmitTransactionRequest_TYPE_ASYNC)
}

func (c *Commander) CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, done func(error)) {
	c.command(ctx, cmd, payload, done, api.SubmitTransactionRequest_TYPE_SYNC)
}

func (c *Commander) command(_ context.Context, cmd txn.Command, payload proto.Message, done func(error), ty api.SubmitTransactionRequest_Type) {
	if c.bc == nil {
		panic("commander was instantiated without a chain")
	}
	go func() {
		ctx, cfunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cfunc()
		inputData := commands.NewInputData(c.bstats.Height())
		wrapPayloadIntoInputData(inputData, cmd, payload)
		marshalledData, err := proto.Marshal(inputData)
		if err != nil {
			// this should never be possible
			c.log.Panic("could not marshal core transaction", logging.Error(err))
		}

		signature, err := c.sign(marshalledData)
		if err != nil {
			// this should never be possible too
			c.log.Panic("could not sign command", logging.Error(err))
		}

		tx := commands.NewTransaction(c.wallet.PubKey().Hex(), marshalledData, signature)
		_, err = c.bc.SubmitTransaction(
			ctx, tx, api.SubmitTransactionRequest_TYPE_SYNC)
		if err != nil {
			// this can happen as network dependent
			c.log.Error("could not send transaction to tendermint",
				logging.Error(err),
				logging.String("tx", payload.String()))
		}

		if done != nil {
			done(err)
		}
	}()
}

func (c *Commander) sign(marshalledData []byte) (*commandspb.Signature, error) {
	sig, err := c.wallet.Sign(marshalledData)
	if err != nil {
		return nil, err
	}

	return commands.NewSignature(sig, c.wallet.Algo(), c.wallet.Version()), nil
}

func wrapPayloadIntoInputData(data *commandspb.InputData, cmd txn.Command, payload proto.Message) {
	switch cmd {
	case txn.SubmitOrderCommand, txn.CancelOrderCommand, txn.AmendOrderCommand, txn.VoteCommand, txn.WithdrawCommand, txn.LiquidityProvisionCommand, txn.ProposeCommand, txn.SubmitOracleDataCommand:
		panic("command is not supported to be sent by a node.")
	case txn.AnnounceNodeCommand:
		if underlyingCmd, ok := payload.(*commandspb.AnnounceNode); ok {
			data.Command = &commandspb.InputData_AnnounceNode{
				AnnounceNode: underlyingCmd,
			}
		} else {
			panic("failed to wrap to AnnounceNode")
		}
	case txn.ValidatorHeartbeatCommand:
		if underlyingCmd, ok := payload.(*commandspb.ValidatorHeartbeat); ok {
			data.Command = &commandspb.InputData_ValidatorHeartbeat{
				ValidatorHeartbeat: underlyingCmd,
			}
		} else {
			panic("failed to wrap to ValidatorHeartbeat")
		}
	case txn.NodeVoteCommand:
		if underlyingCmd, ok := payload.(*commandspb.NodeVote); ok {
			data.Command = &commandspb.InputData_NodeVote{
				NodeVote: underlyingCmd,
			}
		} else {
			panic("failed to wrap to NodeVote")
		}
	case txn.NodeSignatureCommand:
		if underlyingCmd, ok := payload.(*commandspb.NodeSignature); ok {
			data.Command = &commandspb.InputData_NodeSignature{
				NodeSignature: underlyingCmd,
			}
		} else {
			panic("failed to wrap to NodeSignature")
		}
	case txn.ChainEventCommand:
		if underlyingCmd, ok := payload.(*commandspb.ChainEvent); ok {
			data.Command = &commandspb.InputData_ChainEvent{
				ChainEvent: underlyingCmd,
			}
		} else {
			panic("failed to wrap to ChainEvent")
		}
	case txn.CheckpointRestoreCommand:
		if underlyingCmd, ok := payload.(*commandspb.RestoreSnapshot); ok {
			data.Command = &commandspb.InputData_RestoreSnapshotSubmission{
				RestoreSnapshotSubmission: underlyingCmd,
			}
		} else {
			panic("failed to wrap RestoreSnapshot")
		}
	case txn.StateVariableProposalCommand:
		if underlyingCmd, ok := payload.(*commandspb.StateVariableProposal); ok {
			data.Command = &commandspb.InputData_StateVariableProposal{
				StateVariableProposal: underlyingCmd,
			}
		} else {
			panic("failed to wrap StateVariableProposal")
		}
	default:
		panic(fmt.Errorf("command %v is not supported", cmd))
	}
}
