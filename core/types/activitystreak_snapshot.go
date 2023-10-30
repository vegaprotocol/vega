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

type PayloadActivityStreak struct {
	ActivityStreak *snapshotpb.ActivityStreak
}

func (p *PayloadActivityStreak) Key() string {
	return "activitystreak"
}

func (*PayloadActivityStreak) Namespace() SnapshotNamespace {
	return ActivityStreakSnapshot
}

func (p *PayloadActivityStreak) IntoProto() *snapshotpb.Payload_ActivityStreak {
	return &snapshotpb.Payload_ActivityStreak{
		ActivityStreak: p.ActivityStreak,
	}
}

func (*PayloadActivityStreak) isPayload() {}

func (p *PayloadActivityStreak) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadActivityStreakFromProto(vestingPayload *snapshotpb.Payload_ActivityStreak) *PayloadActivityStreak {
	return &PayloadActivityStreak{
		ActivityStreak: vestingPayload.ActivityStreak,
	}
}
