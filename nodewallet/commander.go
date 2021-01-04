package nodewallet

import (
	"context"
	"crypto/rand"
	"math/big"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/proto/gen/golang/api"
	"code.vegaprotocol.io/vega/txn"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet Chain
type Chain interface {
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle, ty api.SubmitTransactionRequest_Type) error
}

type Commander struct {
	bc  Chain
	wal Wallet
}

var (
	ErrCommandMustBeSigned        = errors.New("command requires a signature")
	ErrPayloadNotNodeRegistration = errors.New("expected node registration payload")
	ErrVegaWalletRequired         = errors.New("vega wallet required to start commander")
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
	raw, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	encodedCmd, err := txn.Encode(raw, cmd)
	if err != nil {
		return err
	}

	tx := &types.Transaction{
		InputData: encodedCmd,
		Nonce:     makeNonce(),
		From: &types.Transaction_PubKey{
			PubKey: c.wal.PubKeyOrAddress(),
		},
	}

	rawTx, err := proto.Marshal(tx)
	if err != nil {
		return err
	}

	sig, err := c.wal.Sign(rawTx)
	if err != nil {
		return err
	}

	wrapped := &types.SignedBundle{
		Tx: rawTx,
		Sig: &types.Signature{
			Sig:     sig,
			Algo:    c.wal.Algo(),
			Version: c.wal.Version(),
		},
	}
	return c.bc.SubmitTransaction(ctx, wrapped, api.SubmitTransactionRequest_TYPE_ASYNC)
}

func makeNonce() uint64 {
	max := &big.Int{}
	// set it to the max value of the uint64
	max.SetUint64(^uint64(0))
	nonce, _ := rand.Int(rand.Reader, max)
	return nonce.Uint64()
}
