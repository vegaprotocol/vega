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
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
)

type VolumeDiscountPrograms struct {
	*ConnectionSource
}

func NewVolumeDiscountPrograms(connectionSource *ConnectionSource) *VolumeDiscountPrograms {
	return &VolumeDiscountPrograms{
		ConnectionSource: connectionSource,
	}
}

func (rp *VolumeDiscountPrograms) AddVolumeDiscountProgram(ctx context.Context, program *entities.VolumeDiscountProgram) error {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "AddVolumeDiscountProgram")()
	return rp.insertVolumeDiscountProgram(ctx, program)
}

func (rp *VolumeDiscountPrograms) insertVolumeDiscountProgram(ctx context.Context, program *entities.VolumeDiscountProgram) error {
	if len(program.BenefitTiers) > 0 && program.BenefitTiers[0].TierNumber == nil {
		for i := range program.BenefitTiers {
			program.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
		}
	}
	_, err := rp.Exec(ctx,
		`INSERT INTO volume_discount_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, seq_num)
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

func (rp *VolumeDiscountPrograms) UpdateVolumeDiscountProgram(ctx context.Context, program *entities.VolumeDiscountProgram) error {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "UpdateVolumeDiscountProgram")()
	return rp.insertVolumeDiscountProgram(ctx, program)
}

func (rp *VolumeDiscountPrograms) EndVolumeDiscountProgram(ctx context.Context, version uint64, endedAt time.Time, vegaTime time.Time, seqNum uint64) error {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "EndVolumeDiscountProgram")()
	_, err := rp.Exec(ctx,
		`INSERT INTO volume_discount_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, ended_at, vega_time, seq_num)
            SELECT id, $1, benefit_tiers, end_of_program_timestamp, window_length, $2, $3, $4
            FROM current_volume_discount_program`, version, endedAt, vegaTime, seqNum,
	)

	return err
}

func (rp *VolumeDiscountPrograms) GetCurrentVolumeDiscountProgram(ctx context.Context) (entities.VolumeDiscountProgram, error) {
	var program entities.VolumeDiscountProgram
	defer func() {
		if len(program.BenefitTiers) > 0 && program.BenefitTiers[0].TierNumber == nil {
			for i := range program.BenefitTiers {
				program.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
			}
		}
		metrics.StartSQLQuery("VolumeDiscountPrograms", "GetCurrentVolumeDiscountProgram")()
	}()

	query := `SELECT id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, ended_at, seq_num FROM current_volume_discount_program`
	if err := pgxscan.Get(ctx, rp.ConnectionSource, &program, query); err != nil {
		return program, err
	}

	return program, nil
}
