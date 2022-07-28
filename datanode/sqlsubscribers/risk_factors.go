// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type RiskFactorEvent interface {
	events.Event
	RiskFactor() vega.RiskFactor
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_factor_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/sqlsubscribers RiskFactorStore
type RiskFactorStore interface {
	Upsert(context.Context, *entities.RiskFactor) error
}

type RiskFactor struct {
	subscriber
	store RiskFactorStore
	log   *logging.Logger
}

func NewRiskFactor(store RiskFactorStore, log *logging.Logger) *RiskFactor {
	return &RiskFactor{
		store: store,
		log:   log,
	}
}

func (rf *RiskFactor) Types() []events.Type {
	return []events.Type{events.RiskFactorEvent}
}

func (rf *RiskFactor) Push(ctx context.Context, evt events.Event) error {
	return rf.consume(ctx, evt.(RiskFactorEvent))
}

func (rf *RiskFactor) consume(ctx context.Context, event RiskFactorEvent) error {
	riskFactor := event.RiskFactor()
	record, err := entities.RiskFactorFromProto(&riskFactor, rf.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting risk factor proto to database entity failed")
	}

	return errors.Wrap(rf.store.Upsert(ctx, record), "inserting risk factor to SQL store failed")
}
