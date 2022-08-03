// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/json"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
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

func (w NodeSignature) Cursor() *Cursor {
	cursor := NodeSignatureCursor{
		ID:  w.ResourceID.String(),
		Sig: w.Sig,
	}
	return NewCursor(cursor.String())
}

func (w NodeSignature) ToProtoEdge(_ ...any) (*v2.NodeSignatureEdge, error) {
	return &v2.NodeSignatureEdge{
		Node:   w.ToProto(),
		Cursor: w.Cursor().Encode(),
	}, nil
}

type NodeSignatureCursor struct {
	ID  string `json:"id"`
	Sig []byte `json:"sig"`
}

func (c NodeSignatureCursor) String() string {
	bs, err := json.Marshal(c)
	// Should never error, so panic if it does
	if err != nil {
		panic(fmt.Errorf("could not marshal node signature cursor: %w", err))
	}

	return string(bs)
}

func (c *NodeSignatureCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}
