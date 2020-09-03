package tx

import (
	"errors"

	uuid "github.com/satori/go.uuid"

	"code.vegaprotocol.io/vega/blockchain"
	types "code.vegaprotocol.io/vega/proto"
)

const (
	PrefixLen  = 36
	CommandLen = 1
	HeaderLen  = PrefixLen + CommandLen
	ValidLen   = HeaderLen
)

var (
	ErrInvalidTxPayloadLen = errors.New("payload size is incorrec, should be > 37 bytes")
)

type Tx struct {
	Prefix  string
	Command blockchain.Command
	Input   []byte
}

func NewTx(payload []byte) (*Tx, error) {
	if len(payload) < ValidLen {
		return nil, ErrInvalidTxPayloadLen
	}

	tx := &Tx{
		Prefix:  string(payload[:PrefixLen]),
		Command: blockchain.Command(payload[PrefixLen]),
		Input:   payload[HeaderLen:],
	}
	return tx, nil
}

func NewTxFromProto(tx *types.Transaction) (*Tx, error) {
	return NewTx(tx.InputData)
}

func (tx *Tx) Encode() []byte {
	if tx.Prefix == "" {
		tx.Prefix = uuid.NewV4().String()
	}

	out := append([]byte(tx.Prefix), []byte{byte(tx.Command)}...)
	return append(out, tx.Input...)
}
