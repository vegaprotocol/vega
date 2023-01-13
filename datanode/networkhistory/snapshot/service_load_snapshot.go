package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgtype"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot/orders"
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

type LoadResult struct {
	LoadedFromHeight int64
	LoadedToHeight   int64
	RowsLoaded       int64
}

func (b *Service) LoadAllSnapshotData(ctx context.Context, currentStateSnapshot CurrentState,
	contiguousHistory []History, sourceDir string,
) (LoadResult, error) {
	if err := killAllConnectionsToDatabase(ctx, b.connConfig); err != nil {
		return LoadResult{}, fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	vegaDbConn, err := pgxpool.Connect(context.Background(), b.connConfig.GetConnectionString())
	if err != nil {
		return LoadResult{}, fmt.Errorf("unable to connect to vega database: %w", err)
	}

	datanodeBlockSpan, err := sqlstore.GetDatanodeBlockSpan(ctx, b.connConfig)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to check if datanode has data: %w", err)
	}

	historyFromHeight := contiguousHistory[0].HeightFrom
	historyToHeight := contiguousHistory[len(contiguousHistory)-1].HeightTo

	if err = validateSpanOfHistoryToLoad(datanodeBlockSpan, historyFromHeight, historyToHeight); err != nil {
		return LoadResult{}, err
	}

	heightToLoadFrom := int64(0)
	if datanodeBlockSpan.HasData {
		heightToLoadFrom = datanodeBlockSpan.ToHeight + 1
	} else {
		sqlstore.RevertToSchemaVersionZero(b.log, b.connConfig, sqlstore.EmbedMigrations)
		heightToLoadFrom = historyFromHeight
	}

	_, err = vegaDbConn.Exec(ctx, "SET TIME ZONE 0")
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to set timezone to UTC: %w", err)
	}

	dbVersion, err := getDatabaseVersion(b.connConfig)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to get database version: %w", err)
	}

	b.log.Info("copying data into database", logging.Int64("database version", dbVersion))

	// History first
	var totalRowsCopied int64
	var rowsCopied int64
	var constraints *Constraints
	for _, history := range contiguousHistory {
		if constraints == nil {
			constraints, err = b.removeConstraints(ctx, vegaDbConn)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to remove constraints: %w", err)
			}
		}

		snapshotDatabaseVersion, err := fsutil.GetSnapshotDatabaseVersion(filepath.Join(sourceDir, history.CompressedFileName()))
		if err != nil {
			return LoadResult{}, fmt.Errorf("failed to get snapshot database version: %w", err)
		}

		if dbVersion != snapshotDatabaseVersion {
			b.log.Info("migrating database", logging.Int64("current database version", dbVersion), logging.Int64("target database version", dbVersion))

			err = b.applyConstraints(ctx, vegaDbConn, constraints)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to apply constraints prior to database migration from verion %d to %d: %w", dbVersion, snapshotDatabaseVersion, err)
			}

			err = b.migrateDatabaseToVersion(snapshotDatabaseVersion)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to migrate database from version %d to %d: %w", dbVersion, snapshotDatabaseVersion, err)
			}
			dbVersion = snapshotDatabaseVersion

			constraints, err = b.removeConstraints(ctx, vegaDbConn)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to remove constraints after database migration from version %d to %d: %w", dbVersion, snapshotDatabaseVersion, err)
			}
		}

		rowsCopied, err = b.loadSnapshot(ctx, history, sourceDir, vegaDbConn)
		if err != nil {
			return LoadResult{}, fmt.Errorf("failed to load history snapshot %s: %w", history, err)
		}
		totalRowsCopied += rowsCopied
	}

	// Then current state

	if err = b.truncateCurrentStateTables(ctx, vegaDbConn); err != nil {
		return LoadResult{}, fmt.Errorf("failed to truncate current state tables: %w", err)
	}

	rowsCopied, err = b.loadSnapshot(ctx, currentStateSnapshot, sourceDir, vegaDbConn)
	totalRowsCopied += rowsCopied
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to load current state snapshot %s: %w", currentStateSnapshot, err)
	}

	b.log.Infof("reapplying constraints")
	err = b.applyConstraints(ctx, vegaDbConn, constraints)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to end bulk load: %w", err)
	}

	b.log.Infof("recreating continuous aggregate data")
	err = b.recreateAllContinuousAggregateData(ctx, vegaDbConn)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to recreate continuous aggregate data: %w", err)
	}

	b.log.Infof("restoring current order state")
	err = orders.UpdateCurrentOrdersState(ctx, vegaDbConn)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to update current order state: %w", err)
	}

	return LoadResult{
		LoadedFromHeight: heightToLoadFrom,
		LoadedToHeight:   historyToHeight,
		RowsLoaded:       totalRowsCopied,
	}, nil
}

func validateSpanOfHistoryToLoad(existingDatanodeSpan sqlstore.DatanodeBlockSpan, historyFromHeight int64, historyToHeight int64) error {
	if !existingDatanodeSpan.HasData {
		return nil
	}

	if historyFromHeight < existingDatanodeSpan.FromHeight {
		return fmt.Errorf("loading history from height %d is not possible as it is before the datanodes oldest block height %d, to load this history first empty the datanode",
			historyFromHeight, existingDatanodeSpan.FromHeight)
	}

	if historyFromHeight > existingDatanodeSpan.ToHeight+1 {
		return fmt.Errorf("the from height of the history to load, %d, must fall within or be one greater than the datanodes current span of %d to %d", historyFromHeight,
			existingDatanodeSpan.FromHeight, existingDatanodeSpan.ToHeight)
	}

	if historyFromHeight >= existingDatanodeSpan.FromHeight && historyToHeight <= existingDatanodeSpan.ToHeight {
		return fmt.Errorf("the span of history requested to load, %d to %d, is within the datanodes current span of %d to %d", historyFromHeight,
			historyToHeight, existingDatanodeSpan.FromHeight, existingDatanodeSpan.ToHeight)
	}

	return nil
}

func (b *Service) truncateCurrentStateTables(ctx context.Context, vegaDbConn Conn) error {
	dbMetaData, err := NewDatabaseMetaData(ctx, b.connConfig)
	if err != nil {
		return fmt.Errorf("failed to get database metadata: %w", err)
	}

	for tableName := range dbMetaData.TableNameToMetaData {
		if !dbMetaData.TableNameToMetaData[tableName].Hypertable {
			tableTruncateSQL := fmt.Sprintf("truncate table %s", tableName)
			_, err = vegaDbConn.Exec(ctx, tableTruncateSQL)
			if err != nil {
				return fmt.Errorf("failed to truncate table %s: %w", tableName, err)
			}
		}
	}

	return nil
}

type compressedFileMapping interface {
	UncompressedDataDir() string
	CompressedFileName() string
}

func (b *Service) loadSnapshot(ctx context.Context, snapshotData compressedFileMapping, copyFromDirectory string,
	vegaDbConn *pgxpool.Pool,
) (int64, error) {
	compressedFilePath := filepath.Join(copyFromDirectory, snapshotData.CompressedFileName())
	decompressedFilesDestination := filepath.Join(copyFromDirectory, snapshotData.UncompressedDataDir())
	defer func() {
		_ = os.RemoveAll(compressedFilePath)
		_ = os.RemoveAll(decompressedFilesDestination)
	}()

	err := fsutil.DecompressAndUntarFile(compressedFilePath, decompressedFilesDestination)
	if err != nil {
		return 0, fmt.Errorf("failed to decompress and untar data: %w", err)
	}

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to get database meta data: %w", err)
	}

	historyTableToLastPartitionEntry, err := getLastPartitionEntries(ctx, dbMetaData, vegaDbConn)
	if err != nil {
		return 0, fmt.Errorf("failed to get last partition entries: %w", err)
	}

	b.log.Infof("copying %s into database", snapshotData.UncompressedDataDir())

	rowsCopied, err := b.copyDataIntoDatabase(ctx, vegaDbConn, decompressedFilesDestination,
		filepath.Join(b.snapshotsCopyFromPath, snapshotData.UncompressedDataDir()), dbMetaData, historyTableToLastPartitionEntry)
	if err != nil {
		return 0, fmt.Errorf("failed to copy uncompressed data into the database %s : %w", snapshotData.UncompressedDataDir(), err)
	}

	b.log.Infof("copied %d rows from %s into database", rowsCopied, snapshotData.UncompressedDataDir())

	return rowsCopied, nil
}

func getLastPartitionEntries(ctx context.Context, dbMetaData DatabaseMetadata, vegaDbConn *pgxpool.Pool) (map[string]time.Time, error) {
	historyTableToLastPartitionEntry := map[string]time.Time{}
	for _, historyTableName := range dbMetaData.GetHistoryTableNames() {
		metaData := dbMetaData.TableNameToMetaData[historyTableName]

		lastPartitionEntry, err := getLastPartitionEntryForTable(ctx, vegaDbConn, metaData)
		if err != nil {
			return nil, fmt.Errorf("failed to get last partition entry for table %s: %w", historyTableName, err)
		}

		historyTableToLastPartitionEntry[historyTableName] = lastPartitionEntry
	}

	return historyTableToLastPartitionEntry, nil
}

func getLastPartitionEntryForTable(ctx context.Context, vegaDbConn *pgxpool.Pool, historyTableMetaData TableMetadata) (time.Time, error) {
	timeSelect := fmt.Sprintf(`SELECT %s FROM %s order by %s desc limit 1`, historyTableMetaData.PartitionColumn, historyTableMetaData.Name,
		historyTableMetaData.PartitionColumn)

	rows, err := vegaDbConn.Query(ctx, timeSelect)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query last partition column time for table %s: %w", historyTableMetaData.Name, err)
	}
	defer rows.Close()

	if rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to get values for row: %w", err)
		}
		if len(values) != 1 {
			return time.Time{}, fmt.Errorf("expected just 1 value got %d", len(values))
		}
		partitionTime, ok := values[0].(time.Time)
		if !ok {
			return time.Time{}, fmt.Errorf("expected value to be of type time, got %v", values[0])
		}

		return partitionTime, nil
	}

	return time.Time{}, nil
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
		return fmt.Errorf("failed to kill all database connection: %w", err)
	}

	return nil
}

type Constraints struct {
	indexes              []IndexInfo
	createConstraintsSQL []string
}

func (b *Service) removeConstraints(ctx context.Context, vegaDbConn Conn) (*Constraints, error) {
	// Capture the sql to re-create the constraints
	createContraintRows, err := vegaDbConn.Query(ctx, "SELECT 'ALTER TABLE '||nspname||'.'||relname||' ADD CONSTRAINT '||conname||' '|| pg_get_constraintdef(pg_constraint.oid)||';' "+
		"FROM pg_constraint "+
		"INNER JOIN pg_class ON conrelid=pg_class.oid "+
		"INNER JOIN pg_namespace ON pg_namespace.oid=pg_class.relnamespace where pg_namespace.nspname='public'"+
		"ORDER BY CASE WHEN contype='f' THEN 0 ELSE 1 END DESC,contype DESC,nspname DESC,relname DESC,conname DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to get create constraints sql: %w", err)
	}

	var createConstraintsSQL []string
	for createContraintRows.Next() {
		createConstraintSQL := ""
		err = createContraintRows.Scan(&createConstraintSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan create constraint sql: %w", err)
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
		return nil, fmt.Errorf("failed to get drop constraints sql: %w", err)
	}

	for dropContraintRows.Next() {
		dropConstraintSQL := ""
		err = dropContraintRows.Scan(&dropConstraintSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan drop constraint sql: %w", err)
		}

		_, err = vegaDbConn.Exec(ctx, dropConstraintSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to execute drop constrain %s: %w", dropConstraintSQL, err)
		}
	}

	// Drop all indexes
	var indexes []IndexInfo
	err = pgxscan.Select(ctx, vegaDbConn, &indexes,
		`select tablename, Indexname, Indexdef from pg_indexes where schemaname ='public' order by tablename`)
	if err != nil {
		return nil, fmt.Errorf("failed to get table indexes: %w", err)
	}

	for _, index := range indexes {
		_, err = vegaDbConn.Exec(ctx, fmt.Sprintf("DROP INDEX %s", index.Indexname))
		if err != nil {
			return nil, fmt.Errorf("failed to drop index %s: %w", index.Indexname, err)
		}
	}

	return &Constraints{
		indexes:              indexes,
		createConstraintsSQL: createConstraintsSQL,
	}, nil
}

func (b *Service) copyDataIntoDatabase(ctx context.Context, vegaDbConn *pgxpool.Pool, copyFromDir string,
	databaseCopyFromDir string, dbMetaData DatabaseMetadata, historyTableToLastPartitionEntry map[string]time.Time,
) (int64, error) {
	files, err := os.ReadDir(copyFromDir)
	if err != nil {
		return 0, fmt.Errorf("failed to get files in snapshot dir: %w", err)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Disable all triggers
	_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = replica;")
	if err != nil {
		return 0, fmt.Errorf("failed to disable triggers, setting session replication role to replica failed: %w", err)
	}

	var rowsCopied int64
	for _, file := range files {
		if !file.IsDir() {
			tableName := file.Name()
			snapshotFilePath := filepath.Join(databaseCopyFromDir, tableName)
			tableMetaData := dbMetaData.TableNameToMetaData[tableName]
			if tableMetaData.Hypertable {
				partitionColumn := tableMetaData.PartitionColumn

				var copyQuery string
				if lastPartitionColumnEntry, ok := historyTableToLastPartitionEntry[tableName]; ok {
					timestampString, err := encodeTimestampToString(lastPartitionColumnEntry)
					if err != nil {
						return 0, fmt.Errorf("failed to encode timestamp into string: %w", err)
					}

					copyQuery = fmt.Sprintf(`copy %s from '%s' where %s > timestamp '%s'`, tableName, snapshotFilePath,
						partitionColumn, timestampString)
				} else {
					copyQuery = fmt.Sprintf(`copy %s from '%s'`, tableName, snapshotFilePath)
				}

				tag, err := vegaDbConn.Exec(ctx, copyQuery)
				rowsCopied += tag.RowsAffected()
				if err != nil {
					return 0, fmt.Errorf("failed to copy data into hyper-table %s: %w", tableName, err)
				}
			} else {
				tag, err := vegaDbConn.Exec(ctx, fmt.Sprintf(`copy %s from '%s'`, tableName, snapshotFilePath))
				rowsCopied += tag.RowsAffected()
				if err != nil {
					return 0, fmt.Errorf("failed to copy data into table %s: %w", tableName, err)
				}
			}
		}
	}
	// Enable all triggers
	_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = DEFAULT;")
	if err != nil {
		return 0, fmt.Errorf("failed to enable triggers, setting session replication role to DEFAULT failed: %w", err)
	}

	return rowsCopied, nil
}

// encodeTimestampToString is required as pgx does not support parameter interpolation on copy statements.
func encodeTimestampToString(lastPartitionColumnEntry time.Time) ([]byte, error) {
	lastPartitionColumnEntry = lastPartitionColumnEntry.UTC()

	ts := pgtype.Timestamp{
		Time:   lastPartitionColumnEntry,
		Status: pgtype.Present,
	}

	var err error
	var timeText []byte
	timeText, err = ts.EncodeText(nil, timeText)
	if err != nil {
		return nil, fmt.Errorf("failed to encode timestamp: %w", err)
	}
	return timeText, nil
}

func (b *Service) applyConstraints(ctx context.Context, vegaDbConn Conn, constraints *Constraints) error {
	b.log.Infof("restoring all indexes")
	for _, index := range constraints.indexes {
		_, err := vegaDbConn.Exec(ctx, index.Indexdef)
		if err != nil {
			return fmt.Errorf("failed to drop index %s: %w", index.Indexname, err)
		}
	}

	b.log.Infof("restoring all constraints")
	for _, constraintSQL := range constraints.createConstraintsSQL {
		_, err := vegaDbConn.Exec(ctx, constraintSQL)
		if err != nil {
			return fmt.Errorf("failed to execute create constrain %s: %w", constraints.createConstraintsSQL, err)
		}
	}

	return nil
}

func (b *Service) recreateAllContinuousAggregateData(ctx context.Context, vegaDbConn Conn) error {
	b.log.Infof("recreating all continuous aggregate data")
	continuousAggNameRows, err := vegaDbConn.Query(ctx, "SELECT view_name FROM timescaledb_information.continuous_aggregates;")
	if err != nil {
		return fmt.Errorf("failed to get materialized view names: %w", err)
	}

	for continuousAggNameRows.Next() {
		caggName := ""
		err = continuousAggNameRows.Scan(&caggName)
		if err != nil {
			return fmt.Errorf("failed to scan continuous aggregate Name: %w", err)
		}

		_, err = vegaDbConn.Exec(ctx, fmt.Sprintf("CALL refresh_continuous_aggregate('%s', NULL, NULL);;", caggName))
		if err != nil {
			return fmt.Errorf("failed to refresh continuous aggregate %s: %w", caggName, err)
		}
	}
	return nil
}
