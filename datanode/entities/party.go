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
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type _Party struct{}

type PartyID = ID[_Party]

func NewPartyIDSlice(ids ...string) []PartyID {
	res := make([]PartyID, 0, len(ids))
	for _, v := range ids {
		res = append(res, PartyID(v))
	}
	return res
}

type Party struct {
	ID       PartyID
	TxHash   TxHash
	VegaTime *time.Time // Can be NULL for built-in party 'network'
}

func PartyFromProto(pp *vegapb.Party, txHash TxHash) Party {
	return Party{
		ID:     PartyID(pp.Id),
		TxHash: txHash,
	}
}

func (p Party) ToProto() *vegapb.Party {
	return &vegapb.Party{
		Id: p.ID.String(),
	}
}

func (p Party) Cursor() *Cursor {
	return NewCursor(p.String())
}

func (p Party) ToProtoEdge(_ ...any) (*v2.PartyEdge, error) {
	return &v2.PartyEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}

func (p *Party) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), p)
}

func (p Party) String() string {
	bs, err := json.Marshal(p)
	if err != nil {
		panic(fmt.Errorf("failed to marshal party: %w", err))
	}
	return string(bs)
}

type PartyProfile struct {
	PartyID  PartyID
	Alias    string
	Metadata []*vegapb.Metadata
}

func (p PartyProfile) ToProto() *vegapb.PartyProfile {
	return &vegapb.PartyProfile{
		PartyId:  p.PartyID.String(),
		Alias:    p.Alias,
		Metadata: p.Metadata,
	}
}

func (p PartyProfile) Cursor() *Cursor {
	return NewCursor(p.String())
}

func (p PartyProfile) ToProtoEdge(_ ...any) (*v2.PartyProfileEdge, error) {
	return &v2.PartyProfileEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}

func (p *PartyProfile) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), p)
}

func (p PartyProfile) String() string {
	bs, err := json.Marshal(p)
	if err != nil {
		panic(fmt.Errorf("failed to marshal party profile: %w", err))
	}
	return string(bs)
}

func PartyProfileFromProto(t *vegapb.PartyProfile) *PartyProfile {
	return &PartyProfile{
		PartyID:  PartyID(t.PartyId),
		Alias:    t.Alias,
		Metadata: t.Metadata,
	}
}

type PartyProfileCursor struct {
	ID PartyID
}

func (tc PartyProfileCursor) String() string {
	bs, err := json.Marshal(tc)
	if err != nil {
		panic(fmt.Errorf("could not marshal party profile cursor: %v", err))
	}
	return string(bs)
}

func (tc *PartyProfileCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), tc)
}
