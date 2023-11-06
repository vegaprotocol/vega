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
	"testing"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/stretchr/testify/assert"
)

func TestProposalType_String(t *testing.T) {
	tests := []struct {
		name string
		pt   *v2.ListGovernanceDataRequest_Type
		want string
	}{
		{
			name: "ProposalTypeNewMarket",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_NEW_MARKET),
			want: "newMarket",
		}, {
			name: "ProposalTypeNewAsset",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_NEW_ASSET),
			want: "newAsset",
		}, {
			name: "ProposalTypeUpdateAsset",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_UPDATE_ASSET),
			want: "updateAsset",
		}, {
			name: "ProposalTypeUpdateMarket",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_UPDATE_MARKET),
			want: "updateMarket",
		}, {
			name: "ProposalTypeUpdateNetworkParameter",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_NETWORK_PARAMETERS),
			want: "updateNetworkParameter",
		}, {
			name: "ProposalTypeNewFreeform",
			pt:   toPtr(v2.ListGovernanceDataRequest_TYPE_NEW_FREE_FORM),
			want: "newFreeform",
		}, {
			name: "unknown",
			pt:   toPtr(v2.ListGovernanceDataRequest_Type(100)),
			want: "unknown",
		}, {
			name: "nil",
			pt:   nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, (*ProposalType)(tt.pt).String(), "String()")
		})
	}
}
