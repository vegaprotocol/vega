// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"

	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type cancelTransferResolver VegaResolverRoot

func (r *cancelTransferResolver) TransferID(ctx context.Context, obj *vega.CancelTransfer) (string, error) {
	return obj.Changes.TransferId, nil
}

type newTransferResolver VegaResolverRoot

func (r *newTransferResolver) Source(ctx context.Context, obj *vega.NewTransfer) (string, error) {
	return obj.Changes.Source, nil
}

func (r *newTransferResolver) SourceType(ctx context.Context, obj *vega.NewTransfer) (vega.AccountType, error) {
	return obj.Changes.SourceType, nil
}

func (r *newTransferResolver) Destination(ctx context.Context, obj *vega.NewTransfer) (string, error) {
	return obj.Changes.Destination, nil
}

func (r *newTransferResolver) DestinationType(ctx context.Context, obj *vega.NewTransfer) (vega.AccountType, error) {
	return obj.Changes.SourceType, nil
}

func (r *newTransferResolver) Asset(ctx context.Context, obj *vega.NewTransfer) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Changes.Asset)
}

func (r *newTransferResolver) FractionOfBalance(ctx context.Context, obj *vega.NewTransfer) (string, error) {
	return obj.Changes.FractionOfBalance, nil
}

func (r *newTransferResolver) Amount(ctx context.Context, obj *vega.NewTransfer) (string, error) {
	return obj.Changes.Amount, nil
}

func (r *newTransferResolver) TransferType(ctx context.Context, obj *vega.NewTransfer) (GovernanceTransferType, error) {
	if obj.Changes.TransferType == vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING {
		return GovernanceTransferTypeGovernanceTransferTypeAllOrNothing, nil
	} else {
		return GovernanceTransferTypeGovernanceTransferTypeBestEffort, nil
	}
}

func (r *newTransferResolver) Kind(ctx context.Context, obj *vega.NewTransfer) (GovernanceTransferKind, error) {
	switch obj.Changes.GetKind().(type) {
	case *vega.NewTransferConfiguration_OneOff:
		// Need the concrete type specified in gqlgen.yml, which is vega/events/v1.RecurringTransfer not
		// vega.RecurringTransfer that is in our NewTransfer or else gqlgen won't be able to map it.
		govTransfer := obj.Changes.GetOneOff()
		evtTransfer := &eventspb.OneOffGovernanceTransfer{
			DeliverOn: govTransfer.DeliverOn,
		}
		return evtTransfer, nil
	case *vega.NewTransferConfiguration_Recurring:
		govTransfer := obj.Changes.GetRecurring()
		evtTransfer := &eventspb.RecurringGovernanceTransfer{
			StartEpoch: govTransfer.StartEpoch,
			EndEpoch:   govTransfer.EndEpoch,
		}
		return evtTransfer, nil
	default:
		return nil, ErrUnsupportedTransferKind
	}
}
