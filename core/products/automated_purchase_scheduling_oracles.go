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

package products

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type AutomatedPurhcaseSchedulingOracles struct {
	auctionScheduleSubscriptionID spec.SubscriptionID
	auctionSnapshotSubscriptionID spec.SubscriptionID
	binding                       automatedPurhcaseSchedulingBinding
	auctionScheduleUnsub          spec.Unsubscriber
	auctionSnapshotScheduleUnsub  spec.Unsubscriber
}

type automatedPurhcaseSchedulingBinding struct {
	auctionSnapshotScheduleProperty string
	auctionScheduleProperty         string
	auctionSnapshotScheduleType     datapb.PropertyKey_Type
	auctionScheduleType             datapb.PropertyKey_Type
}

func NewProtocolAutomatedPurchaseScheduleOracle(ctx context.Context, oracleEngine OracleEngine, auctionScheduleSpec, auctionVolumeSnapshotScheduleSpec *datasource.Spec, binding *datasource.SpecBindingForAutomatedPurchase, auctionScheduleCb, auctionVolumeSnapshotScheduleCb spec.OnMatchedData) (*AutomatedPurhcaseSchedulingOracles, error) {
	bind := NewProtocolAutomatedPurchaseBinding(binding)
	asSpec, err := spec.New(*datasource.SpecFromDefinition(*auctionScheduleSpec.Data))
	if err != nil {
		return nil, err
	}
	avssSpec, err := spec.New(*datasource.SpecFromDefinition(*auctionVolumeSnapshotScheduleSpec.Data))
	if err != nil {
		return nil, err
	}

	apOracle := &AutomatedPurhcaseSchedulingOracles{
		binding: bind,
	}

	err = apOracle.bindAll(ctx, oracleEngine, asSpec, avssSpec, auctionScheduleCb, auctionVolumeSnapshotScheduleCb)
	if err != nil {
		return nil, err
	}
	return apOracle, nil
}

func (s *AutomatedPurhcaseSchedulingOracles) bindAll(ctx context.Context, oe OracleEngine, auctionSchedule, auctionVolumeSnapshotSchedule *spec.Spec, auctionScheduleCB, auctionVolumeSnapshotScheduleCB spec.OnMatchedData) error {
	err := s.bindAuctionSchedule(ctx, oe, auctionSchedule, auctionScheduleCB)
	if err != nil {
		return err
	}
	return s.bindAuctionVolumeSnapshotSchedule(ctx, oe, auctionVolumeSnapshotSchedule, auctionVolumeSnapshotScheduleCB)
}

func (s *AutomatedPurhcaseSchedulingOracles) bindAuctionVolumeSnapshotSchedule(ctx context.Context, oe OracleEngine, osForSchedule *spec.Spec, cb spec.OnMatchedData) error {
	err := osForSchedule.EnsureBoundableProperty(s.binding.auctionSnapshotScheduleProperty, s.binding.auctionSnapshotScheduleType)
	if err != nil {
		return fmt.Errorf("invalid  oracle spec binding for schedule data: %w", err)
	}
	if s.auctionSnapshotSubscriptionID, s.auctionSnapshotScheduleUnsub, err = oe.Subscribe(ctx, *osForSchedule, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for schedule data: %w", err)
	}
	return nil
}

func (s *AutomatedPurhcaseSchedulingOracles) bindAuctionSchedule(ctx context.Context, oe OracleEngine, osForSchedule *spec.Spec, cb spec.OnMatchedData) error {
	err := osForSchedule.EnsureBoundableProperty(s.binding.auctionScheduleProperty, s.binding.auctionScheduleType)
	if err != nil {
		return fmt.Errorf("invalid  oracle spec binding for schedule data: %w", err)
	}
	if s.auctionScheduleSubscriptionID, s.auctionScheduleUnsub, err = oe.Subscribe(ctx, *osForSchedule, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for schedule data: %w", err)
	}
	return nil
}

func (s *AutomatedPurhcaseSchedulingOracles) UnsubAll(ctx context.Context) {
	if s.auctionScheduleUnsub != nil {
		s.auctionScheduleUnsub(ctx, s.auctionScheduleSubscriptionID)
		s.auctionScheduleUnsub = nil
	}
	if s.auctionSnapshotScheduleUnsub != nil {
		s.auctionSnapshotScheduleUnsub(ctx, s.auctionSnapshotSubscriptionID)
		s.auctionSnapshotScheduleUnsub = nil
	}
}

func NewProtocolAutomatedPurchaseBinding(p *datasource.SpecBindingForAutomatedPurchase) automatedPurhcaseSchedulingBinding {
	return automatedPurhcaseSchedulingBinding{
		auctionScheduleProperty:         strings.TrimSpace(p.AuctionScheduleProperty),
		auctionSnapshotScheduleProperty: strings.TrimSpace(p.AuctionVolumeSnapshotProperty),
		auctionSnapshotScheduleType:     datapb.PropertyKey_TYPE_TIMESTAMP,
		auctionScheduleType:             datapb.PropertyKey_TYPE_TIMESTAMP,
	}
}
