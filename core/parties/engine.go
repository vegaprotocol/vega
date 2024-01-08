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

package parties

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type Engine struct {
	broker Broker

	// profiles tracks all parties profiles by party ID.
	profiles map[types.PartyID]*types.PartyProfile
}

func (e *Engine) UpdateProfile(ctx context.Context, partyID types.PartyID, cmd *commandspb.UpdatePartyProfile) error {
	if err := e.validateProfileUpdate(partyID, cmd); err != nil {
		return fmt.Errorf("invalid profile update: %w", err)
	}

	profile, exists := e.profiles[partyID]
	if !exists {
		profile = &types.PartyProfile{
			PartyID: partyID,
		}
		e.profiles[partyID] = profile
	}

	profile.Alias = cmd.Alias

	profile.Metadata = map[string]string{}
	for _, m := range cmd.Metadata {
		profile.Metadata[m.Key] = m.Value
	}

	e.notifyProfileUpdate(ctx, profile)

	return nil
}

func (e *Engine) loadPartiesFromSnapshot(partiesPayload *types.PayloadParties) {
	for _, profilePayload := range partiesPayload.Profiles {
		profile := &types.PartyProfile{
			PartyID: types.PartyID(profilePayload.PartyId),
			Alias:   profilePayload.Alias,
		}

		profile.Metadata = map[string]string{}
		for _, m := range profilePayload.Metadata {
			profile.Metadata[m.Key] = m.Value
		}

		e.profiles[profile.PartyID] = profile
	}
}

func (e *Engine) validateProfileUpdate(partyID types.PartyID, cmd *commandspb.UpdatePartyProfile) error {
	if err := e.ensureAliasUniqueness(partyID, cmd.Alias); err != nil {
		return err
	}

	return nil
}

func (e *Engine) ensureAliasUniqueness(partyID types.PartyID, newAlias string) error {
	if newAlias == "" {
		return nil
	}

	for _, profile := range e.profiles {
		if partyID != profile.PartyID && profile.Alias == newAlias {
			return fmt.Errorf("alias %q is already taken", newAlias)
		}
	}

	return nil
}

func (e *Engine) notifyProfileUpdate(ctx context.Context, profile *types.PartyProfile) {
	e.broker.Send(events.NewPartyProfileUpdatedEvent(ctx, profile))
}

func NewEngine(broker Broker) *Engine {
	engine := &Engine{
		broker: broker,

		profiles: map[types.PartyID]*types.PartyProfile{},
	}

	return engine
}
