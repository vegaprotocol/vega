package nodewallet

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/proto/api"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/txn"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet Chain
type Chain interface {
	SubmitTransactionV2(ctx context.Context, tx *commandspb.Transaction, ty api.SubmitTransactionV2Request_Type) error
}

type Commander struct {
	bc  Chain
	wal Wallet
}

var (
	ErrVegaWalletRequired = errors.New("vega wallet required to start commander")
)

// NewCommander - used to sign and send transaction from core
// e.g. NodeRegistration, NodeVote
// chain argument can't be passed in in cmd package, but is used for tests
func NewCommander(bc Chain, wal Wallet) (*Commander, error) {
	if Blockchain(wal.Chain()) != Vega {
		return nil, ErrVegaWalletRequired
	}
	return &Commander{
		bc:  bc,
		wal: wal,
	}, nil
}

// SetChain - currently need to hack around the chicken/egg problem
func (c *Commander) SetChain(bc *blockchain.Client) {
	c.bc = bc
}

// Command - send command to chain
func (c *Commander) Command(ctx context.Context, cmd txn.Command, payload proto.Message) error {
	inputData := commandspb.NewInputData()
	wrapPayloadIntoInputData(inputData, cmd, payload)
	marshalledData, err := proto.Marshal(inputData)
	if err != nil {
		return err
	}

	signature, err := c.sign(marshalledData)
	if err != nil {
		return err
	}

	tx := commandspb.NewTransaction(c.wal.PubKeyOrAddress(), marshalledData, signature)

	return c.bc.SubmitTransactionV2(ctx, tx, api.SubmitTransactionV2Request_TYPE_ASYNC)
}

func (c *Commander) sign(marshalledData []byte) (*commandspb.Signature, error) {
	sig, err := c.wal.Sign(marshalledData)
	if err != nil {
		return nil, err
	}

	return commandspb.NewSignature(sig, c.wal.Algo(), c.wal.Version()), nil
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
