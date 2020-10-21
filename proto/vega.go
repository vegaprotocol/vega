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

// Implement these IsEvent methods so we can used mapped types in GQL union
func (_ Order) IsEvent()          {}
func (_ Account) IsEvent()        {}
func (_ Trade) IsEvent()          {}
func (_ Party) IsEvent()          {}
func (_ MarginLevels) IsEvent()   {}
func (_ MarketData) IsEvent()     {}
func (_ NodeSignature) IsEvent()  {}
func (_ GovernanceData) IsEvent() {}
func (_ RiskFactor) IsEvent()     {}
