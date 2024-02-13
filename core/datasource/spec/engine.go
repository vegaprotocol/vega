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

package spec

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/common"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/core/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// Broker interface. Do not need to mock (use package broker/mock).
type Broker interface {
	Stage(event events.Event)
	Send(event events.Event)
	SendBatch(events []events.Event)
}

// TimeService interface.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/datasource/spec TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// The verifier and filterer need to know about new spec immediately, waiting on the event will lead to spec not found issues
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/spec_activations_listener.go -package mocks code.vegaprotocol.io/vega/core/datasource/spec SpecActivationsListener
type SpecActivationsListener interface {
	OnSpecActivated(context.Context, datasource.Spec) error
	OnSpecDeactivated(context.Context, datasource.Spec)
}

// Engine is responsible for broadcasting the Data to products and risk
// models interested in it.
type Engine struct {
	log                     *logging.Logger
	timeService             TimeService
	broker                  Broker
	subscriptions           *specSubscriptions
	specActivationListeners []SpecActivationsListener
}

// NewEngine creates a new Engine.
func NewEngine(
	log *logging.Logger,
	conf Config,
	ts TimeService,
	broker Broker,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	e := &Engine{
		log:           log,
		timeService:   ts,
		broker:        broker,
		subscriptions: newSpecSubscriptions(),
	}

	return e
}

// ListensToSigners checks if the signatures (pubkeys, ETH addresses) from provided sourcing Data are among the keys
// current Specs listen to.
func (e *Engine) ListensToSigners(data common.Data) bool {
	return e.subscriptions.hasAnySubscribers(func(spec Spec) bool {
		return spec.MatchSigners(data)
	})
}

func (e *Engine) AddSpecActivationListener(listener SpecActivationsListener) {
	e.specActivationListeners = append(e.specActivationListeners, listener)
}

func (e *Engine) HasMatch(data common.Data) (bool, error) {
	result, err := e.subscriptions.filterSubscribers(func(spec Spec) (bool, error) {
		return spec.MatchData(data)
	})
	if err != nil {
		return false, err
	}
	return result.hasMatched(), nil
}

// BroadcastData broadcasts data to products and risk models that are interested
// in it. If no one is listening to this Data, it is discarded.
func (e *Engine) BroadcastData(ctx context.Context, data common.Data) error {
	result, err := e.subscriptions.filterSubscribers(func(spec Spec) (bool, error) {
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
				logging.Strings("signers", common.SignersToStringList(data.Signers)),
				logging.String("data", strings.Join(strs, ", ")),
			)
		}
		return nil
	}
	// we have a match, ensure vega-time meta is set.
	if _, err := data.GetDataTimestampNano(); err != nil {
		if data.MetaData == nil {
			data.MetaData = map[string]string{}
		}
		data.MetaData["vega-time"] = strconv.FormatInt(e.timeService.GetTimeNow().Unix(), 10)
	}

	for _, subscriber := range result.subscribers {
		if err := subscriber(ctx, data); err != nil {
			e.log.Debug("broadcasting data to subscriber failed",
				logging.Error(err),
			)
		}
	}
	e.sendMatchedData(ctx, data, result.specIDs)

	return nil
}

// Subscribe registers a callback for a given Spec that is called when an
// signedoracle Data matches the spec.
// It returns a SubscriptionID that is used to Unsubscribe.
// If cb is nil, the method panics.
func (e *Engine) Subscribe(ctx context.Context, spec Spec, cb OnMatchedData) (SubscriptionID, Unsubscriber, error) {
	if cb == nil {
		panic(fmt.Sprintf("a callback is required for spec %v", spec))
	}
	updatedSubscription, firstSubscription := e.subscriptions.addSubscriber(spec, cb, e.timeService.GetTimeNow())
	if firstSubscription {
		for _, listener := range e.specActivationListeners {
			err := listener.OnSpecActivated(ctx, *spec.OriginalSpec)
			if err != nil {
				return 0, nil, fmt.Errorf("failed to activate spec: %w", err)
			}
		}
	}

	e.sendNewSpecSubscription(ctx, updatedSubscription)
	return updatedSubscription.subscriptionID, func(ctx context.Context, id SubscriptionID) {
		e.Unsubscribe(ctx, id)
	}, nil
}

// Unsubscribe unregisters the callback associated to the SubscriptionID.
// If the id doesn't exist, this method panics.
func (e *Engine) Unsubscribe(ctx context.Context, id SubscriptionID) {
	updatedSubscription, hasNoMoreSubscriber := e.subscriptions.removeSubscriber(id)
	if hasNoMoreSubscriber {
		e.sendSpecDeactivation(ctx, updatedSubscription)
		for _, listener := range e.specActivationListeners {
			listener.OnSpecDeactivated(ctx, updatedSubscription.spec)
		}
	}
}

// sendNewSpecSubscription send an event to the broker to inform of the
// subscription (and thus activation) to a spec.
// This may be a subscription to a brand-new spec, or an additional one.
func (e *Engine) sendNewSpecSubscription(ctx context.Context, update updatedSubscription) {
	proto := &vegapb.ExternalDataSourceSpec{
		Spec: update.spec.IntoProto(),
	}
	proto.Spec.CreatedAt = update.specActivatedAt.UnixNano()
	proto.Spec.Status = vegapb.DataSourceSpec_STATUS_ACTIVE

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.73.13") {
		def := update.spec.GetDefinition()
		switch def.DataSourceType.(type) {
		case ethcallcommon.Spec:
			// save to re-send after upgrade
			// we have to do this because the introduction of the source-chain-id
			// changes the hash of the proto struct, which is used in generation of the ID
			e.broker.Stage(events.NewOracleSpecEvent(ctx, &vegapb.OracleSpec{ExternalDataSourceSpec: proto}))
		}
	}
	e.broker.Send(events.NewOracleSpecEvent(ctx, &vegapb.OracleSpec{ExternalDataSourceSpec: proto}))
}

// sendSpecDeactivation send an event to the broker to inform of
// the deactivation (and thus activation) to a spec.
// This may be a subscription to a brand-new spec, or an additional one.
func (e *Engine) sendSpecDeactivation(ctx context.Context, update updatedSubscription) {
	proto := &vegapb.ExternalDataSourceSpec{
		Spec: update.spec.IntoProto(),
	}

	proto.Spec.CreatedAt = update.specActivatedAt.UnixNano()
	proto.Spec.Status = vegapb.DataSourceSpec_STATUS_DEACTIVATED
	e.broker.Send(events.NewOracleSpecEvent(ctx, &vegapb.OracleSpec{ExternalDataSourceSpec: proto}))
}

// sendMatchedData send an event to the broker to inform of
// a match between a specific data source data and one or several specs.
func (e *Engine) sendMatchedData(ctx context.Context, data common.Data, specIDs []SpecID) {
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

	metaData := make([]*datapb.Property, 0, len(data.MetaData))
	for name, value := range data.MetaData {
		metaData = append(metaData, &datapb.Property{
			Name:  name,
			Value: value,
		})
	}
	sort.Slice(metaData, func(i, j int) bool {
		return strings.Compare(metaData[i].Name, metaData[j].Name) < 0
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
				MetaData:       metaData,
			},
		},
	}
	e.broker.Send(events.NewOracleDataEvent(ctx, vegapb.OracleData{ExternalData: dataProto.ExternalData}))
}
