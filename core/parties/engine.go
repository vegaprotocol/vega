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
	"errors"
	"fmt"
	"slices"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	ErrAliasIsReserved   = errors.New("this alias is reserved")
	ReservedPartyAliases = []string{"network"}
)

type Engine struct {
	broker Broker

	// profiles tracks all parties profiles by party ID.
	profiles                  map[types.PartyID]*types.PartyProfile
	minBalanceToUpdateProfile *num.Uint

	// derivedKeys tracks all derived keys assigned to a party
	derivedKeys map[types.PartyID]map[string]struct{}
}

func (e *Engine) OnMinBalanceForUpdatePartyProfileUpdated(_ context.Context, min *num.Uint) error {
	e.minBalanceToUpdateProfile = min.Clone()
	return nil
}

func (e *Engine) AssignDeriveKey(party types.PartyID, derivedKey string) {
	if _, ok := e.derivedKeys[party]; !ok {
		e.derivedKeys[party] = map[string]struct{}{}
	}
	e.derivedKeys[party][derivedKey] = struct{}{}
}

func (e *Engine) CheckDerivedKeyOwnership(party types.PartyID, derivedKey string) (bool, error) {
	derivedKeys, ok := e.derivedKeys[party]
	if !ok {
		return false, fmt.Errorf("party %q does not have any derived keys", party)
	}

	if _, ok := derivedKeys[derivedKey]; !ok {
		return false, fmt.Errorf("party %q does not own %q", party, derivedKey)
	}

	return true, nil
}

func (e *Engine) CheckSufficientBalanceToUpdateProfile(party types.PartyID, balance *num.Uint) error {
	if balance.LT(e.minBalanceToUpdateProfile) {
		return fmt.Errorf("party %q does not have sufficient balance to update profile code, required balance %s available balance %s", party, e.minBalanceToUpdateProfile.String(), balance.String())
	}
	return nil
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

	if slices.Contains(ReservedPartyAliases, newAlias) {
		return ErrAliasIsReserved
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

		profiles:                  map[types.PartyID]*types.PartyProfile{},
		minBalanceToUpdateProfile: num.UintZero(),
	}

	return engine
}
