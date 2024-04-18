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
	// keys are deprecated, to be removed after 74
	Keys []string
	// Buckets are used with the new upgrade
	Buckets []*snapshot.EventForwarderBucket
}

func (p *PayloadEventForwarder) IntoProto() *snapshot.Payload_EventForwarder {
	return &snapshot.Payload_EventForwarder{
		EventForwarder: &snapshot.EventForwarder{
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
	return EventForwarderSnapshot
}

func PayloadEventForwarderFromProto(ef *snapshot.Payload_EventForwarder) *PayloadEventForwarder {
	return &PayloadEventForwarder{
		Keys:    ef.EventForwarder.AckedEvents,
		Buckets: ef.EventForwarder.Buckets,
	}
}

type PayloadEVMEventForwarders struct {
	EVMEventForwarders []*snapshot.EventForwarder
}

func (p *PayloadEVMEventForwarders) IntoProto() *snapshot.Payload_EvmEventForwarders {
	return &snapshot.Payload_EvmEventForwarders{
		EvmEventForwarders: &snapshot.EVMEventForwarders{
			EvmEventForwarders: p.EVMEventForwarders,
		},
	}
}

func (*PayloadEVMEventForwarders) isPayload() {}

func (p *PayloadEVMEventForwarders) plToProto() interface{} {
	return p.IntoProto()
}

func (*PayloadEVMEventForwarders) Key() string {
	return "all"
}

func (p *PayloadEVMEventForwarders) Namespace() SnapshotNamespace {
	return EVMEventForwardersSnapshot
}

func PayloadEVMEventForwardersFromProto(ef *snapshot.Payload_EvmEventForwarders) *PayloadEVMEventForwarders {
	return &PayloadEVMEventForwarders{
		EVMEventForwarders: ef.EvmEventForwarders.EvmEventForwarders,
	}
}
