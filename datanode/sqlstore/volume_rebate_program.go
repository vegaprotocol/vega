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

package sqlstore

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
)

type VolumeRebatePrograms struct {
	*ConnectionSource
}

func NewVolumeRebatePrograms(connectionSource *ConnectionSource) *VolumeRebatePrograms {
	return &VolumeRebatePrograms{
		ConnectionSource: connectionSource,
	}
}

func (rp *VolumeRebatePrograms) AddVolumeRebateProgram(ctx context.Context, program *entities.VolumeRebateProgram) error {
	defer metrics.StartSQLQuery("VolumeRebatePrograms", "AddVolumeRebateProgram")()
	return rp.insertVolumeRebateProgram(ctx, program)
}

func (rp *VolumeRebatePrograms) insertVolumeRebateProgram(ctx context.Context, program *entities.VolumeRebateProgram) error {
	_, err := rp.Exec(ctx,
		`INSERT INTO volume_rebate_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, seq_num)
    		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		program.ID,
		program.Version,
		program.BenefitTiers,
		program.EndOfProgramTimestamp,
		program.WindowLength,
		program.VegaTime,
		program.SeqNum,
	)
	return err
}

func (rp *VolumeRebatePrograms) UpdateVolumeRebateProgram(ctx context.Context, program *entities.VolumeRebateProgram) error {
	defer metrics.StartSQLQuery("VolumeRebatePrograms", "UpdateVolumeRebateProgram")()
	return rp.insertVolumeRebateProgram(ctx, program)
}

func (rp *VolumeRebatePrograms) EndVolumeRebateProgram(ctx context.Context, version uint64, endedAt time.Time, vegaTime time.Time, seqNum uint64) error {
	defer metrics.StartSQLQuery("VolumeRebatePrograms", "EndVolumeRebateProgram")()
	_, err := rp.Exec(ctx,
		`INSERT INTO volume_rebate_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, ended_at, vega_time, seq_num)
            SELECT id, $1, benefit_tiers, end_of_program_timestamp, window_length, $2, $3, $4
            FROM current_volume_rebate_program`, version, endedAt, vegaTime, seqNum,
	)

	return err
}

func (rp *VolumeRebatePrograms) GetCurrentVolumeRebateProgram(ctx context.Context) (entities.VolumeRebateProgram, error) {
	defer metrics.StartSQLQuery("VolumeRebatePrograms", "GetCurrentVolumeRebateProgram")()
	var programProgram entities.VolumeRebateProgram

	query := `SELECT id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, ended_at, seq_num FROM current_volume_rebate_program`
	if err := pgxscan.Get(ctx, rp.ConnectionSource, &programProgram, query); err != nil {
		return programProgram, err
	}

	return programProgram, nil
}
