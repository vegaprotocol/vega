// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package entities

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type _NodeSignature struct{}

type NodeSignatureID = ID[_NodeSignature]

type NodeSignature struct {
	ResourceID NodeSignatureID
	Sig        []byte
	Kind       NodeSignatureKind
	TxHash     TxHash
	VegaTime   time.Time
}

// packNodeSignatures packs a list signatures into the form form:
// 0x + sig1 + sig2 + ... + sigN in hex encoded form
// If the list is empty, return an empty string instead.
func PackNodeSignatures(signatures []NodeSignature) string {
	pack := ""
	if len(signatures) > 0 {
		pack = "0x"
	}

	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	return pack
}

func NodeSignatureFromProto(ns *commandspb.NodeSignature, txHash TxHash, vegaTime time.Time) (*NodeSignature, error) {
	return &NodeSignature{
		ResourceID: NodeSignatureID(ns.Id),
		Sig:        ns.Sig,
		Kind:       NodeSignatureKind(ns.Kind),
		TxHash:     txHash,
		VegaTime:   vegaTime,
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
		ResourceID: w.ResourceID,
		Sig:        w.Sig,
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
	ResourceID NodeSignatureID `json:"resource_id"`
	Sig        []byte          `json:"sig"`
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
