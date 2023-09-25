package sqlstore

import (
	"context"
	"time"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
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
	_, err := rp.Connection.Exec(ctx,
		`INSERT INTO volume_discount_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, ended_at, seq_num)
    		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		program.ID,
		program.Version,
		program.BenefitTiers,
		program.EndOfProgramTimestamp,
		program.WindowLength,
		program.VegaTime,
		program.EndedAt,
		program.SeqNum,
	)
	return err
}

func (rp *VolumeDiscountPrograms) UpdateVolumeDiscountProgram(ctx context.Context, program *entities.VolumeDiscountProgram) error {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "UpdateVolumeDiscountProgram")()
	return rp.insertVolumeDiscountProgram(ctx, program)
}

func (rp *VolumeDiscountPrograms) EndVolumeDiscountProgram(ctx context.Context, version uint64, vegaTime time.Time, seqNum uint64) error {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "EndVolumeDiscountProgram")()
	_, err := rp.Connection.Exec(ctx,
		`INSERT INTO volume_discount_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, ended_at, seq_num)
            SELECT id, $1, benefit_tiers, end_of_program_timestamp, window_length, $2, $2, $3
            FROM current_volume_discount_program`, version, vegaTime, seqNum,
	)

	return err
}

func (rp *VolumeDiscountPrograms) GetCurrentVolumeDiscountProgram(ctx context.Context) (entities.VolumeDiscountProgram, error) {
	defer metrics.StartSQLQuery("VolumeDiscountPrograms", "GetCurrentVolumeDiscountProgram")()
	var programProgram entities.VolumeDiscountProgram

	query := `SELECT id, version, benefit_tiers, end_of_program_timestamp, window_length, vega_time, ended_at, seq_num FROM current_volume_discount_program`
	if err := pgxscan.Get(ctx, rp.Connection, &programProgram, query); err != nil {
		return programProgram, err
	}

	return programProgram, nil
}
