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
