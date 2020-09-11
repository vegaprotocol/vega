package proto

import (
	proto "github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func NewTxFromSignedBundlePayload(payload []byte) (*Transaction, *SignedBundle, error) {
	bundle := &SignedBundle{}
	if err := proto.Unmarshal(payload, bundle); err != nil {
		return nil, nil, errors.Wrap(err, "unable to unmarshal signed bundle")
	}

	tx := &Transaction{}
	if err := proto.Unmarshal(bundle.Tx, tx); err != nil {
		return nil, nil, errors.Wrap(err, "unable to unmarshal transaction from signed bundle")
	}

	return tx, bundle, nil
}
