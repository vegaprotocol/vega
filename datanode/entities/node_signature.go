// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
