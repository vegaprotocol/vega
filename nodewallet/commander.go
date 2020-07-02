package nodewallet

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet Chain
type Chain interface {
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
	SubmitNodeRegistration(ctx context.Context, reg *types.NodeRegistration) (bool, error)
}

type Commander struct {
	ctx context.Context
	bc  Chain
	wal Wallet
}

var (
	unsigned = map[blockchain.Command]struct{}{}

	ErrCommandMustBeSigned        = errors.New("command requires a signature")
	ErrPayloadNotNodeRegistration = errors.New("expected node registration payload")
	ErrVegaWalletRequired         = errors.New("vega wallet required to start commander")
)

// NewCommander - used to sign and send transaction from core
// e.g. NodeRegistration, NodeVote
// chain argument can't be passed in in cmd package, but is used for tests
func NewCommander(ctx context.Context, bc Chain, wal Wallet) (*Commander, error) {
	if Blockchain(wal.Chain()) != Vega {
		return nil, ErrVegaWalletRequired
	}
	return &Commander{
		ctx: ctx,
		bc:  bc,
		wal: wal,
	}, nil
}

// SetChain - currently need to hack around the chicken/egg problem
func (c *Commander) SetChain(bc *blockchain.Client) {
	c.bc = bc
}

// Command - send command to chain
func (c *Commander) Command(cmd blockchain.Command, payload proto.Message) error {
	raw, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	tx, err := txEncode(raw, cmd)
	if err != nil {
		return err
	}
	sig, err := c.wal.Sign(tx)
	if err != nil {
		return err
	}
	wrapped := &types.SignedBundle{
		Data: tx,
		Sig:  sig,
		Auth: &types.SignedBundle_PubKey{
			PubKey: c.wal.PubKeyOrAddress(),
		},
	}
	_, err = c.bc.SubmitTransaction(c.ctx, wrapped)
	return err
}

func txEncode(input []byte, cmd blockchain.Command) ([]byte, error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}
