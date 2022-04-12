package entities

import (
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
)

type NodeSignatureID struct{ ID }

func NewNodeSignatureID(id string) NodeSignatureID {
	return NodeSignatureID{ID: ID(id)}
}

type NodeSignature struct {
	ResourceID NodeSignatureID
	Sig        []byte
	Kind       NodeSignatureKind
}

func NodeSignatureFromProto(ns *commandspb.NodeSignature) (*NodeSignature, error) {
	return &NodeSignature{
		ResourceID: NewNodeSignatureID(ns.Id),
		Sig:        ns.Sig,
		Kind:       NodeSignatureKind(ns.Kind),
	}, nil
}

func (w NodeSignature) ToProto() *commandspb.NodeSignature {
	return &commandspb.NodeSignature{
		Id:   w.ResourceID.String(),
		Sig:  w.Sig,
		Kind: commandspb.NodeSignatureKind(w.Kind),
	}
}
