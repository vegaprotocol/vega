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

package types

import snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

type PayloadEventForwarder struct {
	Scope string
	// keys are deprecated, to be removed after 74
	Keys []string
	// Buckets are used with the new upgrade
	Buckets []*snapshot.EventForwarderBucket
}

func (p *PayloadEventForwarder) IntoProto() *snapshot.Payload_EventForwarder {
	scope := p.Scope
	if len(scope) == 0 {
		scope = "primary"
	}
	return &snapshot.Payload_EventForwarder{
		EventForwarder: &snapshot.EventForwarder{
			Scope:       scope,
			AckedEvents: p.Keys,
			Buckets:     p.Buckets,
		},
	}
}

func (*PayloadEventForwarder) isPayload() {}

func (p *PayloadEventForwarder) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadEventForwarder) Key() string {
	return "all"
}

func (p *PayloadEventForwarder) Namespace() SnapshotNamespace {
	if p.Scope == "primary" {
		return EventForwarderSnapshot
	}
	return SnapshotNamespace(string(EventForwarderSnapshot) + "." + p.Scope)
}

func PayloadEventForwarderFromProto(ef *snapshot.Payload_EventForwarder) *PayloadEventForwarder {
	scope := ef.EventForwarder.Scope
	if len(scope) == 0 {
		scope = "primary"
	}
	return &PayloadEventForwarder{
		Scope:   scope,
		Keys:    ef.EventForwarder.AckedEvents,
		Buckets: ef.EventForwarder.Buckets,
	}
}
