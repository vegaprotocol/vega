package v1

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/libs/crypto"
)

func NewID(signers []*Signer, filters []*Filter) string {
	buf := []byte{}
	for _, filter := range filters {
		s := filter.Key.Name + filter.Key.Type.String()
		for _, c := range filter.Conditions {
			s += c.Operator.String() + c.Value
		}

		buf = append(buf, []byte(s)...)
	}

	return hex.EncodeToString(crypto.Hash(buf))
}

func (p PropertyKey) DeepClone() *PropertyKey {
	return &PropertyKey{
		Name: p.Name,
		Type: p.Type,
	}
}

func (p Property) DeepClone() *Property {
	return &p
}

func (c Condition) DeepClone() *Condition {
	return &Condition{
		Value:    c.Value,
		Operator: c.Operator,
	}
}

func (s Signer) DeepClone() *Signer {
	return &Signer{
		Signer: s.Signer,
	}
}

func (f Filter) DeepClone() *Filter {
	if f.Key != nil {
		f.Key = f.Key.DeepClone()
	}

	if len(f.Conditions) > 0 {
		conditions := f.Conditions
		f.Conditions = make([]*Condition, len(conditions))
		for i, c := range conditions {
			f.Conditions[i] = c.DeepClone()
		}
	}
	return &f
}
