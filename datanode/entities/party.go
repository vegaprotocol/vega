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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type _Party struct{}

type PartyID = ID[_Party]

type Party struct {
	ID       PartyID
	VegaTime *time.Time // Can be NULL for built-in party 'network'
}

func PartyFromProto(pp *types.Party) Party {
	return Party{ID: PartyID(pp.Id)}
}

func (p *Party) ToProto() *types.Party {
	return &types.Party{Id: p.ID.String()}
}

func (p Party) Cursor() *Cursor {
	return NewCursor(p.VegaTime.In(time.UTC).Format(time.RFC3339Nano))
}

func (p Party) ToProtoEdge(_ ...any) (*v2.PartyEdge, error) {
	return &v2.PartyEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}
