package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestAssetProposalNewAssetDeepClone(t *testing.T) {
	ctx := context.Background()

	p := proto.Proposal{
		Id:        "Id",
		Reference: "Reference",
		PartyId:   "PartyId",
		State:     proto.Proposal_STATE_DECLINED,
		Timestamp: 100000,
		Terms: &proto.ProposalTerms{
			ClosingTimestamp:    2000000,
			EnactmentTimestamp:  3000000,
			ValidationTimestamp: 4000000,
			Change: &proto.ProposalTerms_NewAsset{
				NewAsset: &proto.NewAsset{
					Changes: &proto.AssetDetails{
						Source: &proto.AssetDetails_Erc20{
							Erc20: &proto.ERC20{
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
	p.Id = "Changed"
	p.Reference = "Changed"
	p.PartyId = "Changed"
	p.State = proto.Proposal_STATE_ENACTED
	p.Timestamp = 999
	p.Terms.ClosingTimestamp = 999
	p.Terms.EnactmentTimestamp = 888
	p.Terms.ValidationTimestamp = 777

	na := p.Terms.Change.(*proto.ProposalTerms_NewAsset)
	erc := na.NewAsset.Changes.Source.(*proto.AssetDetails_Erc20)
	erc.Erc20.ContractAddress = "Changed"

	assert.NotEqual(t, p.Id, p2.Id)
	assert.NotEqual(t, p.Reference, p2.Reference)
	assert.NotEqual(t, p.PartyId, p2.PartyId)
	assert.NotEqual(t, p.State, p2.State)
	assert.NotEqual(t, p.Timestamp, p2.Timestamp)

	term := p.Terms
	term2 := p2.Terms
	assert.NotEqual(t, term.ClosingTimestamp, term2.ClosingTimestamp)
	assert.NotEqual(t, term.EnactmentTimestamp, term2.EnactmentTimestamp)
	assert.NotEqual(t, term.ValidationTimestamp, term2.ValidationTimestamp)

	na2 := p2.Terms.Change.(*proto.ProposalTerms_NewAsset)
	erc2 := na2.NewAsset.Changes.Source.(*proto.AssetDetails_Erc20)
	assert.NotEqual(t, erc.Erc20.ContractAddress, erc2.Erc20.ContractAddress)
}
