package ethcall

import (
	"fmt"

	"code.vegaprotocol.io/vega/protos/vega"

	"golang.org/x/crypto/sha3"
)

type Normaliser interface {
	Normalise(callResult []byte) (map[string]string, error)
	ToProto() *vega.EthNormaliser
	Hash() []byte
}

func NormaliserFromProto(proto *vega.EthNormaliser) (Normaliser, error) {
	if proto == nil {
		return nil, fmt.Errorf("trigger proto is nil")
	}

	switch t := proto.Normaliser.(type) {
	case *vega.EthNormaliser_DecimalsNormaliser:
		return DecimalsNormaliserFromProto(t.DecimalsNormaliser), nil
	default:
		return nil, fmt.Errorf("unknown normaliser type: %T", proto.Normaliser)
	}
}

type EthDecimalsNormaliser struct {
	Decimals int64
}

func (t EthDecimalsNormaliser) Normalise(callResult []byte) (map[string]string, error) {
	// TODO - will need access to the abi fragment to interpret the bytes.  Spec should have access to this (its on the call object)
	// Also the spec will contain the property name (e.g. 'price')
	return map[string]string{}, nil
}

func (t EthDecimalsNormaliser) Hash() []byte {
	hashFunc := sha3.New256()
	ident := fmt.Sprintf("ethdecimalsnormaliser: %v", t.Decimals)
	hashFunc.Write([]byte(ident))
	return hashFunc.Sum(nil)
}

func (t EthDecimalsNormaliser) ToProto() *vega.EthNormaliser {
	return &vega.EthNormaliser{
		Normaliser: &vega.EthNormaliser_DecimalsNormaliser{
			DecimalsNormaliser: &vega.EthDecimalsNormaliser{
				Decimals: t.Decimals,
			},
		},
	}
}

func DecimalsNormaliserFromProto(proto *vega.EthDecimalsNormaliser) EthDecimalsNormaliser {
	return EthDecimalsNormaliser{Decimals: proto.Decimals}
}
