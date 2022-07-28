// Copyright (c) 2022 Gobalsky Labs Limited
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

package events_test

import (
	"context"
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/stretchr/testify/assert"
)

func TestAssetProposalNewAssetDeepClone(t *testing.T) {
	ctx := context.Background()

	p := types.Proposal{
		ID:        "Id",
		Reference: "Reference",
		Party:     "PartyId",
		State:     types.ProposalStateDeclined,
		Timestamp: 100000,
		Rationale: &types.ProposalRationale{
			Description: "Wen moon!",
			Hash:        "0xdeadbeef",
			URL:         "example.com",
		},
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    2000000,
			EnactmentTimestamp:  3000000,
			ValidationTimestamp: 4000000,
			Change: &types.ProposalTermsNewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetDetails{
						Source: &types.AssetDetailsErc20{
							ERC20: &types.ERC20{
								ContractAddress: "Address",
							},
						},
					},
				},
			},
		},
	}

	proposalEvent := events.NewProposalEvent(ctx, p)
	p2 := proposalEvent.Proposal()

	// Change the original and check we are not updating the wrapped event
	p.ID = "Changed"
	p.Reference = "Changed"
	p.Party = "Changed"
	p.State = types.ProposalStateEnacted
	p.Timestamp = 999
	p.Terms.ClosingTimestamp = 999
	p.Terms.EnactmentTimestamp = 888
	p.Terms.ValidationTimestamp = 777
	p.Rationale.Description = "Wen mars!"
	p.Rationale.Hash = "oxcafed00d"
	p.Rationale.URL = "www.example.com/mars"

	na := p.Terms.Change.(*types.ProposalTermsNewAsset)
	erc := na.NewAsset.Changes.Source.(*types.AssetDetailsErc20)
	erc.ERC20.ContractAddress = "Changed"

	assert.NotEqual(t, p.ID, p2.Id)
	assert.NotEqual(t, p.Reference, p2.Reference)
	assert.NotEqual(t, p.Party, p2.PartyId)
	assert.NotEqual(t, p.State, p2.State)
	assert.NotEqual(t, p.Timestamp, p2.Timestamp)

	term := p.Terms
	term2 := p2.Terms
	assert.NotEqual(t, term.ClosingTimestamp, term2.ClosingTimestamp)
	assert.NotEqual(t, term.EnactmentTimestamp, term2.EnactmentTimestamp)
	assert.NotEqual(t, term.ValidationTimestamp, term2.ValidationTimestamp)

	rationale := p.Rationale
	rationale2 := p2.Rationale
	assert.NotEqual(t, rationale.Description, rationale2.Description)
	assert.NotEqual(t, rationale.Hash, rationale2.Hash)
	assert.NotEqual(t, rationale.URL, rationale2.Url)

	na2 := p2.Terms.Change.(*proto.ProposalTerms_NewAsset)
	erc2 := na2.NewAsset.Changes.Source.(*proto.AssetDetails_Erc20)
	assert.NotEqual(t, erc.ERC20.ContractAddress, erc2.Erc20.ContractAddress)
}
