package nodewallet

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/protos/commands"
	"code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	commanderNamedLogger = "commander"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet Chain
type Chain interface {
	SubmitTransactionV2(ctx context.Context, tx *commandspb.Transaction, ty api.SubmitTransactionV2Request_Type) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_stats_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet BlockchainStats
type BlockchainStats interface {
	Height() uint64
}

type Commander struct {
	log    *logging.Logger
	bc     Chain
	wal    Wallet
	bstats BlockchainStats
}

var (
	ErrVegaWalletRequired = errors.New("vega wallet required to start commander")
)

// NewCommander - used to sign and send transaction from core
// e.g. NodeRegistration, NodeVote
// chain argument can't be passed in in cmd package, but is used for tests
func NewCommander(log *logging.Logger, bc Chain, wal Wallet, bstats BlockchainStats) (*Commander, error) {
	log = log.Named(commanderNamedLogger)
	if Blockchain(wal.Chain()) != Vega {
		return nil, ErrVegaWalletRequired
	}
	return &Commander{
		log:    log,
		bc:     bc,
		wal:    wal,
		bstats: bstats,
	}, nil
}

// SetChain - currently need to hack around the chicken/egg problem
func (c *Commander) SetChain(bc *blockchain.Client) {
	c.bc = bc
}

// Command - send command to chain
func (c *Commander) Command(ctx context.Context, cmd txn.Command, payload proto.Message, done func(bool)) {
	go func() {
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

		tx := commands.NewTransaction(c.wal.PubKeyOrAddress().Hex(), marshalledData, signature)
		err = c.bc.SubmitTransactionV2(ctx, tx, api.SubmitTransactionV2Request_TYPE_ASYNC)
		if err != nil {
			// this can happen as network dependent
			c.log.Error("could not send transaction to tendermint",
				logging.Error(err),
				logging.String("tx", payload.String()))
		}

		if done != nil {
			done(err == nil)
		}
	}()
}

func (c *Commander) sign(marshalledData []byte) (*commandspb.Signature, error) {
	sig, err := c.wal.Sign(marshalledData)
	if err != nil {
		return nil, err
	}

	return commands.NewSignature(sig, c.wal.Algo(), c.wal.Version()), nil
}

func wrapPayloadIntoInputData(data *commandspb.InputData, cmd txn.Command, payload proto.Message) {
	switch cmd {
	case txn.SubmitOrderCommand, txn.CancelOrderCommand, txn.AmendOrderCommand, txn.VoteCommand, txn.WithdrawCommand, txn.LiquidityProvisionCommand, txn.ProposeCommand, txn.SubmitOracleDataCommand:
		panic("command is not supported to be sent by a node.")
	case txn.RegisterNodeCommand:
		if underlyingCmd, ok := payload.(*commandspb.NodeRegistration); ok {
			data.Command = &commandspb.InputData_NodeRegistration{
				NodeRegistration: underlyingCmd,
			}
		} else {
			panic("failed to wrap to NodeRegistration")
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
	default:
		panic(fmt.Errorf("command %v is not supported", cmd))
	}
}
