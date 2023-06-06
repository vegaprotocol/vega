package types

import (
	"errors"
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"golang.org/x/crypto/sha3"
)

type normaliser interface {
	isNormaliser()
	oneOfProto() interface{}
	String() string
	Normalise(callResult []byte) (map[string]string, error)
	Hash() []byte
}

type EthDecimalsNormaliser struct {
	Decimals int64
}

func (e *EthDecimalsNormaliser) isNormaliser() {}

func (e *EthDecimalsNormaliser) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *EthDecimalsNormaliser) IntoProto() *vegapb.EthDecimalsNormaliser {
	return &vegapb.EthDecimalsNormaliser{
		Decimals: e.Decimals,
	}
}

func (e *EthDecimalsNormaliser) String() string {
	return fmt.Sprintf("decimals(%d)", e.Decimals)
}

func (e *EthDecimalsNormaliser) Normalise(callResult []byte) (map[string]string, error) {
	// TODO - will need access to the abi fragment to interpret the bytes.  Spec should have access to this (its on the call object)
	// Also the spec will contain the property name (e.g. 'price')
	return map[string]string{}, nil
}

func (e *EthDecimalsNormaliser) Hash() []byte {
	hashFunc := sha3.New256()
	ident := fmt.Sprintf("ethdecimalsnormaliser: %v", e.Decimals)
	hashFunc.Write([]byte(ident))
	return hashFunc.Sum(nil)
}

func EthDecimalsNormaliserFromProto(protoNormaliser *vegapb.EthDecimalsNormaliser) *EthDecimalsNormaliser {
	if protoNormaliser != nil {
		return &EthDecimalsNormaliser{
			Decimals: protoNormaliser.Decimals,
		}
	}

	return &EthDecimalsNormaliser{}
}

type NormaliserEthDecimalsNormaliser struct {
	EthDecimalsNormaliser *EthDecimalsNormaliser
}

func (n *NormaliserEthDecimalsNormaliser) isNormaliser() {}

func (n *NormaliserEthDecimalsNormaliser) oneOfProto() interface{} {
	return n.IntoProto()
}

func (n *NormaliserEthDecimalsNormaliser) IntoProto() *vegapb.Normaliser_EthDecimalsNormaliser {
	if n.EthDecimalsNormaliser != nil {
		return &vegapb.Normaliser_EthDecimalsNormaliser{
			EthDecimalsNormaliser: n.EthDecimalsNormaliser.IntoProto(),
		}
	}

	return &vegapb.Normaliser_EthDecimalsNormaliser{}
}

func (n *NormaliserEthDecimalsNormaliser) String() string {
	dn := ""
	if n.EthDecimalsNormaliser != nil {
		dn = n.EthDecimalsNormaliser.String()
	}

	return fmt.Sprintf("normaliserethdecimalsnormaliser(%s)", dn)
}

func (n *NormaliserEthDecimalsNormaliser) Normalise(callResult []byte) (map[string]string, error) {
	if n.EthDecimalsNormaliser != nil {
		return n.EthDecimalsNormaliser.Normalise(callResult)
	}

	return nil, errors.New("")
}

func (n *NormaliserEthDecimalsNormaliser) Hash() []byte {
	if n.EthDecimalsNormaliser != nil {
		return n.EthDecimalsNormaliser.Hash()
	}

	return nil
}

func NormaliserEthDecimalsNormaliserFromProto(protoNormaliser *vegapb.Normaliser_EthDecimalsNormaliser) (*NormaliserEthDecimalsNormaliser, error) {
	if protoNormaliser != nil {
		if protoNormaliser.EthDecimalsNormaliser != nil {
			return &NormaliserEthDecimalsNormaliser{
				EthDecimalsNormaliser: &EthDecimalsNormaliser{
					Decimals: protoNormaliser.EthDecimalsNormaliser.Decimals,
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("missing eth decimals normaliser")
}

type Normaliser struct {
	Normaliser normaliser
}

func (n *Normaliser) isNormaliser() {}

func (n *Normaliser) oneOfProto() interface{} {
	return n.IntoProto()
}

func (n *Normaliser) IntoProto() *vegapb.Normaliser {
	if n.Normaliser != nil {
		switch tp := n.Normaliser.(type) {
		case *NormaliserEthDecimalsNormaliser:
			return &vegapb.Normaliser{
				Normaliser: tp.IntoProto(),
			}
		}
	}

	return &vegapb.Normaliser{}
}

func (n *Normaliser) String() string {
	ns := ""
	if n.Normaliser != nil {
		switch tp := n.Normaliser.(type) {
		case *EthDecimalsNormaliser:
			ns = tp.String()
		}
	}

	return fmt.Sprintf("normaliser(%s)", ns)
}

func (n *Normaliser) Normalise(callResult []byte) (map[string]string, error) {
	if n.Normaliser != nil {
		return n.Normaliser.Normalise(callResult)
	}

	return nil, errors.New("")
}

func (n *Normaliser) Hash() []byte {
	if n.Normaliser != nil {
		return n.Normaliser.Hash()
	}

	return nil
}

func NormaliserFromProto(protoNormaliser *vegapb.Normaliser) (*Normaliser, error) {
	if protoNormaliser != nil {
		if protoNormaliser.Normaliser != nil {
			switch tp := protoNormaliser.Normaliser.(type) {
			case *vegapb.Normaliser_EthDecimalsNormaliser:
				return &Normaliser{
					Normaliser: EthDecimalsNormaliserFromProto(tp.EthDecimalsNormaliser),
				}, nil
			}
		}
	}

	return &Normaliser{}, nil
}
