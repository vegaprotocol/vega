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

package oracles

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// Broker interface. Do not need to mock (use package broker/mock).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// TimeService interface.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/oracles TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// The verifier and filterer need to know about new spec immediately, waiting on the event will lead to spec not found issues
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/spec_activations_listener.go -package mocks code.vegaprotocol.io/vega/core/oracles SpecActivationsListener
type SpecActivationsListener interface {
	OnSpecActivated(context.Context, types.OracleSpec) error
	OnSpecDeactivated(context.Context, types.OracleSpec)
}

// Engine is responsible for broadcasting the OracleData to products and risk
// models interested in it.
type Engine struct {
	log                    *logging.Logger
	timeService            TimeService
	broker                 Broker
	subscriptions          *specSubscriptions
	specActivationListener SpecActivationsListener
}

// NewEngine creates a new oracle Engine.
func NewEngine(
	log *logging.Logger,
	conf Config,
	ts TimeService,
	broker Broker,
	specActivationListeners SpecActivationsListener,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	e := &Engine{
		log:                    log,
		timeService:            ts,
		broker:                 broker,
		subscriptions:          newSpecSubscriptions(),
		specActivationListener: specActivationListeners,
	}

	return e
}

// ListensToSigners checks if the signatures (pubkeys, ETH addresses) from provided OracleData are among the keys
// current OracleSpecs listen to.
func (e *Engine) ListensToSigners(data OracleData) bool {
	return e.subscriptions.hasAnySubscribers(func(spec OracleSpec) bool {
		return spec.MatchSigners(data)
	})
}

func (e *Engine) HasMatch(data OracleData) (bool, error) {
	result, err := e.subscriptions.filterSubscribers(func(spec OracleSpec) (bool, error) {
		return spec.MatchData(data)
	})
	if err != nil {
		return false, err
	}
	return result.hasMatched(), nil
}

// BroadcastData broadcasts data to products and risk models that are interested
// in it. If no one is listening to this OracleData, it is discarded.
func (e *Engine) BroadcastData(ctx context.Context, data OracleData) error {
	result, err := e.subscriptions.filterSubscribers(func(spec OracleSpec) (bool, error) {
		return spec.MatchData(data)
	})
	if err != nil {
		e.log.Debug("error in filtering subscribers",
			logging.Error(err),
		)
		return err
	}

	if !result.hasMatched() {
		if e.log.IsDebug() {
			strs := make([]string, 0, len(data.Data))
			for k, v := range data.Data {
				strs = append(strs, fmt.Sprintf("%s:%s", k, v))
			}
			e.log.Debug(
				"no subscriber matches the oracle data",
				logging.Strings("signers", types.SignersToStringList(data.Signers)),
				logging.String("data", strings.Join(strs, ", ")),
			)
		}
		return nil
	}

	for _, subscriber := range result.subscribers {
		if err := subscriber(ctx, data); err != nil {
			e.log.Debug("broadcasting data to subscriber failed",
				logging.Error(err),
			)
		}
	}
	e.sendMatchedOracleData(ctx, data, result.oracleSpecIDs)

	return nil
}

// Subscribe registers a callback for a given OracleSpec that is called when an
// OracleData matches the spec.
// It returns a SubscriptionID that is used to Unsubscribe.
// If cb is nil, the method panics.
func (e *Engine) Subscribe(ctx context.Context, spec OracleSpec, cb OnMatchedOracleData) (SubscriptionID, Unsubscriber, error) {
	if cb == nil {
		panic(fmt.Sprintf("a callback is required for spec %v", spec))
	}
	updatedSubscription, firstSubscription := e.subscriptions.addSubscriber(spec, cb, e.timeService.GetTimeNow())
	if firstSubscription {
		err := e.specActivationListener.OnSpecActivated(ctx, *spec.OriginalSpec)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to activate spec: %w", err)
		}
	}

	e.sendNewOracleSpecSubscription(ctx, updatedSubscription)
	return updatedSubscription.subscriptionID, func(ctx context.Context, id SubscriptionID) {
		e.Unsubscribe(ctx, id)
	}, nil
}

// Unsubscribe unregisters the callback associated to the SubscriptionID.
// If the id doesn't exist, this method panics.
func (e *Engine) Unsubscribe(ctx context.Context, id SubscriptionID) {
	updatedSubscription, hasNoMoreSubscriber := e.subscriptions.removeSubscriber(id)
	if hasNoMoreSubscriber {
		e.sendOracleSpecDeactivation(ctx, updatedSubscription)
		e.specActivationListener.OnSpecDeactivated(ctx, updatedSubscription.spec)
	}
}

// sendNewOracleSpecSubscription send an event to the broker to inform of the
// subscription (and thus activation) to an oracle spec.
// This may be a subscription to a brand-new oracle spec, or an additional one.
func (e *Engine) sendNewOracleSpecSubscription(ctx context.Context, update updatedSubscription) {
	proto := &vegapb.ExternalDataSourceSpec{
		Spec: &vegapb.DataSourceSpec{},
	}
	if update.spec.ExternalDataSourceSpec != nil {
		proto = update.spec.ExternalDataSourceSpec.IntoProto()
	}
	proto.Spec.CreatedAt = update.specActivatedAt.UnixNano()
	proto.Spec.Status = vegapb.DataSourceSpec_STATUS_ACTIVE
	e.broker.Send(events.NewOracleSpecEvent(ctx, vegapb.OracleSpec{ExternalDataSourceSpec: proto}))
}

// sendOracleSpecDeactivation send an event to the broker to inform of
// the deactivation (and thus activation) to an oracle spec.
// This may be a subscription to a brand-new oracle spec, or an additional one.
func (e *Engine) sendOracleSpecDeactivation(ctx context.Context, update updatedSubscription) {
	proto := &vegapb.ExternalDataSourceSpec{
		Spec: &vegapb.DataSourceSpec{},
	}
	if update.spec.ExternalDataSourceSpec != nil {
		proto = update.spec.ExternalDataSourceSpec.IntoProto()
	}
	proto.Spec.CreatedAt = update.specActivatedAt.UnixNano()
	proto.Spec.Status = vegapb.DataSourceSpec_STATUS_DEACTIVATED
	e.broker.Send(events.NewOracleSpecEvent(ctx, vegapb.OracleSpec{ExternalDataSourceSpec: proto}))
}

// sendMatchedOracleData send an event to the broker to inform of
// a match between an oracle data and one or several oracle specs.
func (e *Engine) sendMatchedOracleData(ctx context.Context, data OracleData, specIDs []OracleSpecID) {
	payload := make([]*datapb.Property, 0, len(data.Data))
	for name, value := range data.Data {
		payload = append(payload, &datapb.Property{
			Name:  name,
			Value: value,
		})
	}

	sort.Slice(payload, func(i, j int) bool {
		return strings.Compare(payload[i].Name, payload[j].Name) < 0
	})

	ids := make([]string, 0, len(specIDs))
	for _, specID := range specIDs {
		ids = append(ids, string(specID))
	}

	sigs := make([]*datapb.Signer, len(data.Signers))
	for i, s := range data.Signers {
		sigs[i] = s.IntoProto()
	}

	dataProto := vegapb.OracleData{
		ExternalData: &datapb.ExternalData{
			Data: &datapb.Data{
				Signers:        sigs,
				Data:           payload,
				MatchedSpecIds: ids,
				BroadcastAt:    e.timeService.GetTimeNow().UnixNano(),
			},
		},
	}
	e.broker.Send(events.NewOracleDataEvent(ctx, vegapb.OracleData{ExternalData: dataProto.ExternalData}))
}
