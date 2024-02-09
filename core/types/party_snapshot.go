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

package types

import snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

type PayloadParties struct {
	Profiles []*snapshotpb.PartyProfile
}

func (p *PayloadParties) Key() string {
	return "parties"
}

func (*PayloadParties) Namespace() SnapshotNamespace {
	return PartiesSnapshot
}

func (p *PayloadParties) IntoProto() *snapshotpb.Payload_Parties {
	return &snapshotpb.Payload_Parties{
		Parties: &snapshotpb.Parties{
			Profiles: p.Profiles,
		},
	}
}

func (*PayloadParties) isPayload() {}

func (p *PayloadParties) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadPartiesFromProto(payload *snapshotpb.Payload_Parties) *PayloadParties {
	return &PayloadParties{
		Profiles: payload.Parties.GetProfiles(),
	}
}
