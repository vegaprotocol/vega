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

type PayloadVesting struct {
	Vesting *snapshotpb.Vesting
}

func (p *PayloadVesting) Key() string {
	return "vesting"
}

func (*PayloadVesting) Namespace() SnapshotNamespace {
	return VestingSnapshot
}

func (p *PayloadVesting) IntoProto() *snapshotpb.Payload_Vesting {
	return &snapshotpb.Payload_Vesting{
		Vesting: p.Vesting,
	}
}

func (*PayloadVesting) isPayload() {}

func (p *PayloadVesting) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadVestingFromProto(vestingPayload *snapshotpb.Payload_Vesting) *PayloadVesting {
	return &PayloadVesting{
		Vesting: vestingPayload.Vesting,
	}
}
