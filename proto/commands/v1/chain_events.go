package v1

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

func (c *ChainEvent) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(c.TxId)...)
	out = append(out, []byte(fmt.Sprintf("%v", c.Nonce))...)

	var (
		next []byte
		err  error
	)
	switch evt := c.Event.(type) {
	case *ChainEvent_Erc20:
		next, err = evt.Erc20.PrepareToSign()
	case *ChainEvent_Builtin:
		next, err = evt.Builtin.PrepareToSign()
	default:
		err = types.ErrUnsupportedChainEvent
	}

	if err != nil {
		return nil, err
	}

	out = append(out, next...)
	return out, nil
}
