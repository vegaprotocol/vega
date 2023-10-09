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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	_ReferralSet  struct{}
	ReferralSetID = ID[_ReferralSet]

	ReferralSet struct {
		ID        ReferralSetID
		Referrer  PartyID
		CreatedAt time.Time
		UpdatedAt time.Time
		VegaTime  time.Time
	}

	ReferralSetReferee struct {
		ReferralSetID ReferralSetID
		Referee       PartyID
		JoinedAt      time.Time
		AtEpoch       uint64
		VegaTime      time.Time
	}

	ReferralSetCursor struct {
		CreatedAt time.Time
		ID        ReferralSetID
	}

	ReferralSetRefereeCursor struct {
		JoinedAt time.Time
		Referee  PartyID
	}
)

func ReferralSetFromProto(proto *eventspb.ReferralSetCreated, vegaTime time.Time) *ReferralSet {
	return &ReferralSet{
		ID:        ReferralSetID(proto.SetId),
		Referrer:  PartyID(proto.Referrer),
		CreatedAt: time.Unix(0, proto.CreatedAt),
		UpdatedAt: time.Unix(0, proto.UpdatedAt),
		VegaTime:  vegaTime,
	}
}

func ReferralSetRefereeFromProto(proto *eventspb.RefereeJoinedReferralSet, vegaTime time.Time) *ReferralSetReferee {
	return &ReferralSetReferee{
		ReferralSetID: ReferralSetID(proto.SetId),
		Referee:       PartyID(proto.Referee),
		JoinedAt:      time.Unix(0, proto.JoinedAt),
		AtEpoch:       proto.AtEpoch,
		VegaTime:      vegaTime,
	}
}

func (rs ReferralSet) ToProto() *v2.ReferralSet {
	return &v2.ReferralSet{
		Id:        rs.ID.String(),
		Referrer:  rs.Referrer.String(),
		CreatedAt: rs.CreatedAt.UnixNano(),
		UpdatedAt: rs.UpdatedAt.UnixNano(),
	}
}

func (rs ReferralSet) Cursor() *Cursor {
	c := ReferralSetCursor{
		CreatedAt: rs.CreatedAt,
		ID:        rs.ID,
	}
	return NewCursor(c.ToString())
}

func (rs ReferralSet) ToProtoEdge(_ ...any) (*v2.ReferralSetEdge, error) {
	return &v2.ReferralSetEdge{
		Node:   rs.ToProto(),
		Cursor: rs.Cursor().Encode(),
	}, nil
}

func (r ReferralSetReferee) ToProto() *v2.ReferralSetReferee {
	return &v2.ReferralSetReferee{
		ReferralSetId: r.ReferralSetID.String(),
		Referee:       r.Referee.String(),
		JoinedAt:      r.JoinedAt.UnixNano(),
		AtEpoch:       r.AtEpoch,
	}
}

func (r ReferralSetReferee) Cursor() *Cursor {
	c := ReferralSetRefereeCursor{
		JoinedAt: r.JoinedAt,
		Referee:  r.Referee,
	}
	return NewCursor(c.ToString())
}

func (r ReferralSetReferee) ToProtoEdge(_ ...any) (*v2.ReferralSetRefereeEdge, error) {
	return &v2.ReferralSetRefereeEdge{
		Node:   r.ToProto(),
		Cursor: r.Cursor().Encode(),
	}, nil
}

func (c ReferralSetCursor) ToString() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal referral set cursor: %v", err))
	}
	return string(bs)
}

func (c *ReferralSetCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

func (c ReferralSetRefereeCursor) ToString() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal referral set referee cursor: %v", err))
	}
	return string(bs)
}

func (c *ReferralSetRefereeCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
