// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
