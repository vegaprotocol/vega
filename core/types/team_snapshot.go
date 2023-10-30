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

import snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

type PayloadTeams struct {
	Teams []*snapshotpb.Team
}

func (p *PayloadTeams) Key() string {
	return "teams"
}

func (*PayloadTeams) Namespace() SnapshotNamespace {
	return TeamsSnapshot
}

func (p *PayloadTeams) IntoProto() *snapshotpb.Payload_Teams {
	return &snapshotpb.Payload_Teams{
		Teams: &snapshotpb.Teams{
			Teams: p.Teams,
		},
	}
}

func (*PayloadTeams) isPayload() {}

func (p *PayloadTeams) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadTeamsFromProto(teamsPayload *snapshotpb.Payload_Teams) *PayloadTeams {
	return &PayloadTeams{
		Teams: teamsPayload.Teams.GetTeams(),
	}
}

type PayloadTeamSwitches struct {
	TeamSwitches []*snapshotpb.TeamSwitch
}

func (p *PayloadTeamSwitches) Key() string {
	return "teamSwitches"
}

func (*PayloadTeamSwitches) Namespace() SnapshotNamespace {
	return TeamsSnapshot
}

func (p *PayloadTeamSwitches) IntoProto() *snapshotpb.Payload_TeamSwitches {
	return &snapshotpb.Payload_TeamSwitches{
		TeamSwitches: &snapshotpb.TeamSwitches{
			TeamSwitches: p.TeamSwitches,
		},
	}
}

func (*PayloadTeamSwitches) isPayload() {}

func (p *PayloadTeamSwitches) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadTeamSwitchesFromProto(teamsPayload *snapshotpb.Payload_TeamSwitches) *PayloadTeamSwitches {
	return &PayloadTeamSwitches{
		TeamSwitches: teamsPayload.TeamSwitches.GetTeamSwitches(),
	}
}
