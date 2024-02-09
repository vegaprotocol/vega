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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type (
	FundingPeriodEvent interface {
		events.Event
		FundingPeriod() *eventspb.FundingPeriod
	}
	FundingPeriodDataPointEvent interface {
		events.Event
		FundingPeriodDataPoint() *eventspb.FundingPeriodDataPoint
	}
	FundingPeriodStore interface {
		AddFundingPeriod(ctx context.Context, period *entities.FundingPeriod) error
		AddDataPoint(ctx context.Context, dataPoint *entities.FundingPeriodDataPoint) error
	}
	FundingPeriod struct {
		subscriber
		store FundingPeriodStore
	}
)

func NewFundingPeriod(store FundingPeriodStore) *FundingPeriod {
	return &FundingPeriod{
		store: store,
	}
}

func (fp *FundingPeriod) Types() []events.Type {
	return []events.Type{
		events.FundingPeriodEvent,
		events.FundingPeriodDataPointEvent,
	}
}

func (fp *FundingPeriod) Push(ctx context.Context, evt events.Event) error {
	switch evt.Type() {
	case events.FundingPeriodEvent:
		return fp.consumeFundingPeriodEvent(ctx, evt.(FundingPeriodEvent))
	case events.FundingPeriodDataPointEvent:
		return fp.consumeFundingPeriodDataPointEvent(ctx, evt.(FundingPeriodDataPointEvent))
	default:
		return nil
	}
}

func (fp *FundingPeriod) consumeFundingPeriodEvent(ctx context.Context, evt FundingPeriodEvent) error {
	fundingPeriod, err := entities.NewFundingPeriodFromProto(evt.FundingPeriod(), entities.TxHash(evt.TxHash()), fp.vegaTime)
	if err != nil {
		return errors.Wrap(err, "deserializing funding period")
	}
	return errors.Wrap(fp.store.AddFundingPeriod(ctx, fundingPeriod), "adding funding period")
}

func (fp *FundingPeriod) consumeFundingPeriodDataPointEvent(ctx context.Context, evt FundingPeriodDataPointEvent) error {
	dataPoint, err := entities.NewFundingPeriodDataPointFromProto(evt.FundingPeriodDataPoint(), entities.TxHash(evt.TxHash()), fp.vegaTime)
	if err != nil {
		return errors.Wrap(err, "deserializing funding period data point")
	}
	return errors.Wrap(fp.store.AddDataPoint(ctx, dataPoint), "adding funding period data point")
}
