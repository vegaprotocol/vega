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

package store

import (
	"context"
	"fmt"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Migrator is a data migration agent that will migrate tx_results data
// that is required for providing data for the block explorer APIs.
// The tx_results table is populated by Tendermint and the number of records
// can be very large. This causes data migration difficult when there is a need
// to add additional information to tx_results in order to fulfill requirements
// for the block explorer APIs.
// The migration agent will allow migrations to run in the background without
// blocking upgrades with slow database migration scripts.
type Migrator struct {
	pool        *pgxpool.Pool
	migrateData bool
}

// NewMigrator creates a new data migration agent.
func NewMigrator(pool *pgxpool.Pool, migrateData bool) *Migrator {
	return &Migrator{
		pool:        pool,
		migrateData: migrateData,
	}
}

func (m *Migrator) checkCanMigrate() bool {
	// We only want to migrate if we have a tx_results_old table
	sql := `select table_name from information_schema.tables where table_name = 'tx_results_old'`
	var tableName string
	if err := m.pool.QueryRow(context.Background(), sql).Scan(&tableName); err != nil {
		return false
	}
	return true
}

func (m *Migrator) cleanupOldData() error {
	// we want to drop the old table if it exists
	sql := `drop table if exists tx_results_old`
	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return fmt.Errorf("could not drop old table: %w", err)
	}

	return nil
}

// Migrate will run the data migration.
func (m *Migrator) Migrate() error {
	if !m.checkCanMigrate() {
		return nil
	}

	// create indexes on the tables that we will be querying
	if err := m.createIndexes(); err != nil {
		return err
	}

	// get a list of dates that we need to migrate
	migrateDates, err := m.getMigrationDates()
	if err != nil {
		return err
	}

	// loop through each date and migrate the data for the date
	for _, d := range migrateDates {
		// we're going to make use of temporary tables which are only visible to the session that created them
		// we therefore have to use a single connection and make sure that the same connection is used for the
		// migration process.
		conn, err := m.pool.Acquire(context.Background())
		if err != nil {
			return fmt.Errorf("could not acquire connection: %w", err)
		}
		// if we error, we want to stop the migration rather than continue as we do
		if err := m.doMigration(conn, d); err != nil {
			return fmt.Errorf("could not migrate data for date %s: %w", d.Format("2006-01-02"), err)
		}
		// make sure we release the connection back to the pool when we're done
		conn.Release()
	}

	if err := m.cleanupOldData(); err != nil {
		return fmt.Errorf("could not drop redundant migration data: %w", err)
	}

	return nil
}

func (m *Migrator) getMigrationDates() ([]time.Time, error) {
	sql := `create table if not exists migration_dates(
		migration_date date primary key,
		migrated bool default (false)
	)`

	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return nil, fmt.Errorf("could not create migration_dates table: %w", err)
	}

	// now let's populate the data we need, only new dates that aren't in the table will be added
	sql = `insert into migration_dates(migration_date)
		select distinct created_at::date
		from blocks
		on conflict do nothing`

	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return nil, fmt.Errorf("could not populate migration_dates table: %w", err)
	}

	// now retrieve the dates that we need to migrate in reverse order because we want to migrate the latest
	// data first
	sql = `select migration_date from migration_dates where migrated = false order by migration_date desc`

	var migrationDates []struct {
		MigrationDate time.Time
	}

	if err := pgxscan.Select(context.Background(), m.pool, &migrationDates, sql); err != nil {
		return nil, fmt.Errorf("could not retrieve migration dates: %w", err)
	}

	dates := make([]time.Time, len(migrationDates))

	for i, d := range migrationDates {
		dates[i] = d.MigrationDate
	}

	return dates, nil
}

func (m *Migrator) doMigration(conn *pgxpool.Conn, date time.Time) error {
	startDate := date
	endDate := date.AddDate(0, 0, 1)

	// pre-migration cleanup
	cleanupSQL := []string{
		`drop table if exists blocks_temp`,
		`drop table if exists tx_results_temp`,
	}

	for _, sql := range cleanupSQL {
		if _, err := conn.Exec(context.Background(), sql); err != nil {
			return fmt.Errorf("could not cleanup temporary tables: %w", err)
		}
	}

	// create a temporary table for the blocks that need to be migrated for the given date
	migrateSQL := []struct {
		SQL  string
		args []any
	}{
		{
			// just get the blocks we need to update for the date
			SQL:  `select * into blocks_temp from blocks where created_at >= $1 and created_at < $2`,
			args: []any{startDate, endDate},
		},
		{
			// and the tx_results for the date
			SQL:  `select * into tx_results_temp from tx_results_old where created_at >= $1 and created_at < $2`,
			args: []any{startDate, endDate},
		},
		{
			// create an index on the temporary blocks table
			SQL:  `create index idx_blocks_temp_rowid on blocks_temp(rowid)`,
			args: []any{},
		},
		{
			// create an index on the temporary tx_results table
			SQL:  `create index idx_tx_results_temp_block_id on tx_results_temp(block_id)`,
			args: []any{},
		},
		{
			// update the tx_results_temp table with the block height for the date
			SQL: `update tx_results_temp t
				set block_height = b.height
			from blocks_temp b
			where t.block_id = b.rowid`,
			args: []any{},
		},
		{
			// now insert this date's data into the tx_results table
			SQL: `insert into tx_results(rowid, block_id, index, created_at, tx_hash, tx_result, submitter, cmd_type, block_height)
			select rowid, block_id, index, created_at, tx_hash, tx_result, submitter, cmd_type, block_height
			from tx_results_temp`,
			args: []any{},
		},
		// now drop the temporary tables
		{
			SQL:  `drop table if exists blocks_temp`,
			args: []any{},
		},
		{
			SQL:  `drop table if exists tx_results_temp`,
			args: []any{},
		},
		// and update migration dates to show that we've migrated this date
		{
			SQL:  `update migration_dates set migrated = true where migration_date = $1`,
			args: []any{date},
		},
	}

	for _, query := range migrateSQL {
		if _, err := conn.Exec(context.Background(), query.SQL, query.args...); err != nil {
			return fmt.Errorf("could not migrate data for date %s: %w", date.Format("2006-01-02"), err)
		}
	}

	return nil
}

func (m *Migrator) createIndexes() error {
	sql := `create index if not exists idx_tx_results_old_created_at on tx_results_old(created_at)`
	// this index creation could take some time, but we don't know how long it should take so we don't want to timeout
	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return fmt.Errorf("could not create created_at index for tx_results_old: %w", err)
	}

	sql = `create index if not exists idx_blocks_created_at on blocks(created_at)`
	// this index creation could take some time, but we don't know how long it should take so we don't want to timeout
	if _, err := m.pool.Exec(context.Background(), sql); err != nil {
		return fmt.Errorf("could not create created_at index for blocks: %w", err)
	}

	return nil
}
