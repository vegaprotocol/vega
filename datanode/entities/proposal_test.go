package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
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
