// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nodewallets

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/nodewallets/vega"
	"code.vegaprotocol.io/vega/core/txn"
	"code.vegaprotocol.io/vega/logging"
	api "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cenkalti/backoff"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/golang/protobuf/proto"
)

const (
	commanderNamedLogger = "commander"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/core/nodewallets Chain
type Chain interface {
	SubmitTransactionSync(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultBroadcastTx, error)
	SubmitTransactionAsync(ctx context.Context, tx *commandspb.Transaction) (*tmctypes.ResultBroadcastTx, error)
	GetChainID(ctx context.Context) (string, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_stats_mock.go -package mocks code.vegaprotocol.io/vega/core/nodewallets BlockchainStats
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

// Command - send command to chain.
// Note: beware when passing in an exponential back off since the done function may be called many times.
func (c *Commander) Command(ctx context.Context, cmd txn.Command, payload proto.Message, done func(string, error), bo *backoff.ExponentialBackOff) {
	c.command(ctx, cmd, payload, done, api.SubmitTransactionRequest_TYPE_SYNC, bo, nil)
}

func (c *Commander) CommandSync(ctx context.Context, cmd txn.Command, payload proto.Message, done func(string, error), bo *backoff.ExponentialBackOff) {
	c.command(ctx, cmd, payload, done, api.SubmitTransactionRequest_TYPE_SYNC, bo, nil)
}

func (c *Commander) CommandWithPoW(ctx context.Context, cmd txn.Command, payload proto.Message, done func(string, error), bo *backoff.ExponentialBackOff, pow *commandspb.ProofOfWork) {
	c.command(ctx, cmd, payload, done, api.SubmitTransactionRequest_TYPE_SYNC, bo, pow)
}

func (c *Commander) command(ctx context.Context, cmd txn.Command, payload proto.Message, done func(string, error), ty api.SubmitTransactionRequest_Type, bo *backoff.ExponentialBackOff, pow *commandspb.ProofOfWork) {
	if c.bc == nil {
		panic("commander was instantiated without a chain")
	}
	f := func() error {
		innerCtx, cfunc := context.WithTimeout(ctx, 5*time.Second)
		defer cfunc()

		chainID, err := c.bc.GetChainID(innerCtx)
		if err != nil {
			c.log.Error("couldn't retrieve chain ID",
				logging.Error(err),
			)
			return err
		}

		inputData := commands.NewInputData(c.bstats.Height())
		wrapPayloadIntoInputData(inputData, cmd, payload)
		marshalInputData, err := commands.MarshalInputData(inputData)
		if err != nil {
			// this should never be possible
			c.log.Panic("could not marshal core transaction", logging.Error(err))
		}

		signature, err := c.sign(commands.BundleInputDataForSigning(marshalInputData, chainID))
		if err != nil {
			// this should never be possible too
			c.log.Panic("could not sign command", logging.Error(err))
		}

		tx := commands.NewTransaction(c.wallet.PubKey().Hex(), marshalInputData, signature)
		tx.Pow = pow

		var resp *tmctypes.ResultBroadcastTx
		if ty == api.SubmitTransactionRequest_TYPE_SYNC {
			resp, err = c.bc.SubmitTransactionSync(innerCtx, tx)
		} else {
			resp, err = c.bc.SubmitTransactionAsync(innerCtx, tx)
		}

		var txHash string
		switch {
		case err != nil:
			c.log.Error("could not send transaction to tendermint",
				logging.Error(err),
				logging.String("tx", payload.String()))
		case resp.Code != 0:
			err = fmt.Errorf("%s", string(resp.Data.Bytes()))
			c.log.Error("transaction reached network but was rejected",
				logging.Error(err))
		default:
			txHash = resp.Hash.String()
		}

		if done != nil {
			done(txHash, err)
		}

		return err
	}

	if bo != nil {
		go backoff.Retry(f, bo)
	} else {
		go f()
	}
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
	case txn.SubmitOrderCommand, txn.CancelOrderCommand, txn.AmendOrderCommand, txn.VoteCommand, txn.WithdrawCommand, txn.LiquidityProvisionCommand, txn.ProposeCommand, txn.SubmitOracleDataCommand, txn.StopOrdersCancellationCommand, txn.StopOrdersSubmissionCommand:
		panic("command is not supported to be sent by a node.")
	case txn.ProtocolUpgradeCommand:
		if underlyingCmd, ok := payload.(*commandspb.ProtocolUpgradeProposal); ok {
			data.Command = &commandspb.InputData_ProtocolUpgradeProposal{
				ProtocolUpgradeProposal: underlyingCmd,
			}
		} else {
			panic("failed to wrap to ProtocolUpgradeProposal")
		}
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
	case txn.StateVariableProposalCommand:
		if underlyingCmd, ok := payload.(*commandspb.StateVariableProposal); ok {
			data.Command = &commandspb.InputData_StateVariableProposal{
				StateVariableProposal: underlyingCmd,
			}
		} else {
			panic("failed to wrap StateVariableProposal")
		}
	case txn.RotateEthereumKeySubmissionCommand:
		if underlyingCmd, ok := payload.(*commandspb.EthereumKeyRotateSubmission); ok {
			data.Command = &commandspb.InputData_EthereumKeyRotateSubmission{
				EthereumKeyRotateSubmission: underlyingCmd,
			}
		} else {
			panic("failed to wrap RotateEthereumKeySubmissionCommand")
		}
	default:
		panic(fmt.Errorf("command %v is not supported", cmd))
	}
}
