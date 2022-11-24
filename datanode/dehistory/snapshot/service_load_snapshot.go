package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/datanode/dehistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot/orders"
	"code.vegaprotocol.io/vega/logging"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (b *Service) LoadAllSnapshotData(ctx context.Context, currentStateSnapshot CurrentState,
	contiguousHistory []History, sourceDir string,
) (int64, error) {
	if err := killAllConnectionsToDatabase(ctx, b.connConfig); err != nil {
		return 0, fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	vegaDbConn, err := pgxpool.Connect(context.Background(), b.connConfig.GetConnectionString())
	if err != nil {
		return 0, fmt.Errorf("unable to connect to vega database:%w", err)
	}

	_, err = vegaDbConn.Exec(ctx, "SET TIME ZONE 0")
	if err != nil {
		return 0, fmt.Errorf("failed to set timezone to UTC:%w", err)
	}

	b.log.Infof("preparing for bulk load")
	indexes, createConstrainsSQL, err := b.beforeBulkLoad(ctx, vegaDbConn)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare database for bulk load: %w", err)
	}

	dbVersion, err := getDatabaseVersion(b.connConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to get database version:%w", err)
	}

	b.log.Info("copying data into database", logging.Int64("database version", dbVersion))
	// History first
	var totalRowsCopied int64
	var rowsCopied int64
	for _, history := range contiguousHistory {
		rowsCopied, dbVersion, err = b.loadSnapshot(ctx, history, sourceDir, dbVersion, vegaDbConn)
		totalRowsCopied += rowsCopied
		if err != nil {
			return 0, fmt.Errorf("failed to load history snapshot %s: %w", history, err)
		}
	}

	// Then current state
	rowsCopied, _, err = b.loadSnapshot(ctx, currentStateSnapshot, sourceDir, dbVersion, vegaDbConn)
	totalRowsCopied += rowsCopied
	if err != nil {
		return 0, fmt.Errorf("failed to load current state snapshot %s: %w", currentStateSnapshot, err)
	}

	b.log.Infof("preparing database")
	err = b.afterBulkLoad(ctx, vegaDbConn, indexes, createConstrainsSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to end bulk load: %w", err)
	}

	b.log.Infof("restoring current order state")
	err = orders.UpdateCurrentOrdersState(ctx, vegaDbConn)
	if err != nil {
		return 0, fmt.Errorf("failed to update current order state: %w", err)
	}

	return totalRowsCopied, nil
}

type compressedFileMapping interface {
	UncompressedDataDir() string
	CompressedFileName() string
}

func (b *Service) loadSnapshot(ctx context.Context, snapshotData compressedFileMapping, copyFromDirectory string, currentDbVersion int64,
	vegaDbConn *pgxpool.Pool,
) (int64, int64, error) {
	compressedFilePath := filepath.Join(copyFromDirectory, snapshotData.CompressedFileName())
	decompressedFilesDestination := filepath.Join(copyFromDirectory, snapshotData.UncompressedDataDir())
	defer func() {
		_ = os.RemoveAll(compressedFilePath)
		_ = os.RemoveAll(decompressedFilesDestination)
	}()

	snapshotDbVersion, err := fsutil.DecompressAndUntarFile(compressedFilePath, decompressedFilesDestination)
	if err != nil {
		return 0, currentDbVersion, fmt.Errorf("failed to decompress and untar data: %w", err)
	}

	if currentDbVersion != snapshotDbVersion {
		err = b.migrateDatabaseToVersion(snapshotDbVersion)
		if err != nil {
			return 0, currentDbVersion, fmt.Errorf("failed to migrate database from version %d to %d: %w", currentDbVersion, snapshotDbVersion, err)
		}
		currentDbVersion = snapshotDbVersion
	}

	b.log.Infof("copying %s into database", snapshotData.UncompressedDataDir())

	rowsCopied, err := b.copyDataIntoDatabase(ctx, vegaDbConn, decompressedFilesDestination,
		filepath.Join(b.snapshotsCopyFromPath, snapshotData.UncompressedDataDir()))
	if err != nil {
		return 0, currentDbVersion, fmt.Errorf("failed to copy uncompressed data into the database %s : %w", snapshotData.UncompressedDataDir(), err)
	}

	b.log.Infof("copied %d rows from %s into database", rowsCopied, snapshotData.UncompressedDataDir())

	return rowsCopied, currentDbVersion, nil
}

func killAllConnectionsToDatabase(ctx context.Context, connConfig sqlstore.ConnectionConfig) error {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	killAllConnectionsQuery := fmt.Sprintf(
		`SELECT
	pg_terminate_backend(pg_stat_activity.pid)
		FROM
	pg_stat_activity
		WHERE
	pg_stat_activity.datname = '%s'
	AND pid <> pg_backend_pid();`, connConfig.Database)

	_, err = conn.Exec(ctx, killAllConnectionsQuery)
	if err != nil {
		return fmt.Errorf("failed to kill all database connection:%w", err)
	}

	return nil
}

func (b *Service) beforeBulkLoad(ctx context.Context, vegaDbConn Conn) (indexes []IndexInfo, createConstraintsSQL []string, err error) {
	// Capture the sql to re-create the constraints
	createContraintRows, err := vegaDbConn.Query(ctx, "SELECT 'ALTER TABLE '||nspname||'.'||relname||' ADD CONSTRAINT '||conname||' '|| pg_get_constraintdef(pg_constraint.oid)||';' "+
		"FROM pg_constraint "+
		"INNER JOIN pg_class ON conrelid=pg_class.oid "+
		"INNER JOIN pg_namespace ON pg_namespace.oid=pg_class.relnamespace where pg_namespace.nspname='public'"+
		"ORDER BY CASE WHEN contype='f' THEN 0 ELSE 1 END DESC,contype DESC,nspname DESC,relname DESC,conname DESC")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get create constraints sql:%w", err)
	}

	for createContraintRows.Next() {
		createConstraintSQL := ""
		err = createContraintRows.Scan(&createConstraintSQL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan create constraint sql:%w", err)
		}

		createConstraintsSQL = append(createConstraintsSQL, createConstraintSQL)
	}

	// Drop all constraints
	dropContraintRows, err := vegaDbConn.Query(ctx, "SELECT 'ALTER TABLE '||nspname||'.'||relname||' DROP CONSTRAINT '||conname||';'"+
		"FROM pg_constraint "+
		"INNER JOIN pg_class ON conrelid=pg_class.oid "+
		"INNER JOIN pg_namespace ON pg_namespace.oid=pg_class.relnamespace where pg_namespace.nspname='public' "+
		"ORDER BY CASE WHEN contype='f' THEN 0 ELSE 1 END,contype,nspname,relname,conname")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get drop constraints sql:%w", err)
	}

	for dropContraintRows.Next() {
		dropConstraintSQL := ""
		err = dropContraintRows.Scan(&dropConstraintSQL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan drop constraint sql:%w", err)
		}

		_, err = vegaDbConn.Exec(ctx, dropConstraintSQL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to execute drop constrain %s: %w", dropConstraintSQL, err)
		}
	}

	// Drop all indexes
	err = pgxscan.Select(ctx, vegaDbConn, &indexes,
		`select tablename, Indexname, Indexdef from pg_indexes where schemaname ='public' order by tablename`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get table indexes:%w", err)
	}

	for _, index := range indexes {
		_, err = vegaDbConn.Exec(ctx, fmt.Sprintf("DROP INDEX %s", index.Indexname))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to drop index %s: %w", index.Indexname, err)
		}
	}

	return indexes, createConstraintsSQL, nil
}

func (b *Service) copyDataIntoDatabase(ctx context.Context, vegaDbConn *pgxpool.Pool, copyFromDir string,
	databaseCopyFromDir string,
) (int64, error) {
	files, err := os.ReadDir(copyFromDir)
	if err != nil {
		return 0, fmt.Errorf("failed to get files in snapshot dir:%w", err)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to connect to database:%w", err)
	}

	// Disable all triggers
	_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = replica;")
	if err != nil {
		return 0, fmt.Errorf("failed to disable triggers, setting session replication role to replica failed:%w", err)
	}

	var rowsCopied int64
	for _, file := range files {
		if !file.IsDir() {
			snapshotFilePath := filepath.Join(databaseCopyFromDir, file.Name())
			tag, err := vegaDbConn.Exec(ctx, fmt.Sprintf(`copy %s from '%s'`, file.Name(), snapshotFilePath))
			rowsCopied += tag.RowsAffected()

			if err != nil {
				return 0, fmt.Errorf("failed to copy data into table %s: %w", file.Name(), err)
			}
		}
	}
	// Enable all triggers
	_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = DEFAULT;")
	if err != nil {
		return 0, fmt.Errorf("failed to enable triggers, setting session replication role to DEFAULT failed:%w", err)
	}

	return rowsCopied, nil
}

func (b *Service) afterBulkLoad(ctx context.Context, vegaDbConn Conn, indexes []IndexInfo, createConstraintsSQL []string) error {
	b.log.Infof("restoring all indexes")
	for _, index := range indexes {
		_, err := vegaDbConn.Exec(ctx, index.Indexdef)
		if err != nil {
			return fmt.Errorf("failed to drop index %s: %w", index.Indexname, err)
		}
	}

	b.log.Infof("restoring all constraints")
	for _, constraintSQL := range createConstraintsSQL {
		_, err := vegaDbConn.Exec(ctx, constraintSQL)
		if err != nil {
			return fmt.Errorf("failed to execute create constrain %s: %w", createConstraintsSQL, err)
		}
	}

	b.log.Infof("recreating all continuous aggregate data")
	continuousAggNameRows, err := vegaDbConn.Query(ctx, "SELECT view_name FROM timescaledb_information.continuous_aggregates;")
	if err != nil {
		return fmt.Errorf("failed to get materialized view names:%w", err)
	}

	for continuousAggNameRows.Next() {
		caggName := ""
		err = continuousAggNameRows.Scan(&caggName)
		if err != nil {
			return fmt.Errorf("failed to scan continuous aggregate Name:%w", err)
		}

		_, err = vegaDbConn.Exec(ctx, fmt.Sprintf("CALL refresh_continuous_aggregate('%s', NULL, NULL);;", caggName))
		if err != nil {
			return fmt.Errorf("failed to refresh continuous aggregate %s:%w", caggName, err)
		}
	}

	return nil
}
