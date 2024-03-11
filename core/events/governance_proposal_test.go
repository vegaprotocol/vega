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

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"

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
			Title:       "0xdeadbeef",
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
								ChainID:         "1",
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
	p.Rationale.Title = "oxcafed00d"

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
	assert.NotEqual(t, rationale.Title, rationale2.Title)

	na2 := p2.Terms.Change.(*proto.ProposalTerms_NewAsset)
	erc2 := na2.NewAsset.Changes.Source.(*proto.AssetDetails_Erc20)
	assert.NotEqual(t, erc.ERC20.ContractAddress, erc2.Erc20.ContractAddress)
}
