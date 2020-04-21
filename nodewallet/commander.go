package nodewallet

import (
	"context"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chain_mock.go -package mocks code.vegaprotocol.io/vega/wallet Chain
type Chain interface {
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
	SubmitNodeRegistration(ctx context.Context, reg *types.NodeRegistration) (bool, error)
}

type Commander struct {
	ctx context.Context
	bc  Chain
}

var (
	unsigned = map[blockchain.Command]struct{}{
		blockchain.RegisterNodeCommand: {},
	}

	ErrCommandMustBeSigned        = errors.New("command requires a signature")
	ErrPayloadNotNodeRegistration = errors.New("expected node registration payload")
)

// NewCommander - used to sign and send transaction from core
// e.g. NodeRegistration, NodeVote
// chain argument can't be passed in in cmd package, but is used for tests
func NewCommander(ctx context.Context, bc Chain) *Commander {
	return &Commander{
		ctx: ctx,
		bc:  bc,
	}
}

// SetChain - currently need to hack around the chicken/egg problem
func (c *Commander) SetChain(bc *blockchain.Client) {
	c.bc = bc
}

// Command - send command to chain
func (c *Commander) Command(key Wallet, cmd blockchain.Command, payload proto.Message) error {
	if _, ok := unsigned[cmd]; key == nil && !ok {
		return ErrCommandMustBeSigned
	}
	if key == nil {
		reg, ok := payload.(*types.NodeRegistration)
		if !ok {
			return ErrPayloadNotNodeRegistration
		}
		if _, err := c.bc.SubmitNodeRegistration(c.ctx, reg); err != nil {
			return err
		}
		return nil
	}
	raw, err := proto.Marshal(payload)
	if err != nil {
		return err
	}
	tx, err := txEncode(raw, cmd)
	if err != nil {
		return err
	}
	sig, err := key.Sign(tx)
	if err != nil {
		return err
	}
	wrapped := &types.SignedBundle{
		Data: tx,
		Sig:  sig,
		Auth: &types.SignedBundle_PubKey{
			PubKey: key.PubKeyOrAddress(),
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
