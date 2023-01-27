package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/logging"

	"go.uber.org/zap"

	"github.com/jackc/pgtype"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot/orders"
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

type LoadLog interface {
	Infof(s string, args ...interface{})
	Info(msg string, fields ...zap.Field)
}

func (b *Service) LoadSnapshotData(ctx context.Context, log LoadLog, currentStateSnapshot CurrentState,
	contiguousHistory []History, sourceDir string, connConfig sqlstore.ConnectionConfig, withIndexesAndOrderTriggers bool,
) (LoadResult, error) {
	datanodeBlockSpan, err := sqlstore.GetDatanodeBlockSpan(ctx, b.connPool)
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
		sqlstore.RevertToSchemaVersionZero(b.log, connConfig, sqlstore.EmbedMigrations)
		heightToLoadFrom = historyFromHeight
	}

	_, err = b.connPool.Exec(ctx, "SET TIME ZONE 0")
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to set timezone to UTC: %w", err)
	}

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connPool)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to get database meta data: %w", err)
	}

	log.Info("copying data into database", logging.Int64("database version", dbMetaData.DatabaseVersion))

	// History first
	var totalRowsCopied int64
	var rowsCopied int64
	var removedConstraintsAndIndexes *constraintsAndIndexes
	for _, history := range contiguousHistory {
		if removedConstraintsAndIndexes == nil {
			constraintsSQL, err := b.removeConstraints(ctx, log, b.connPool)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to remove constraints: %w", err)
			}

			indexes, err := b.dropAllIndexes(ctx, log, b.connPool, withIndexesAndOrderTriggers)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to drop all indexes: %w", err)
			}

			removedConstraintsAndIndexes = &constraintsAndIndexes{createConstraintsSQL: constraintsSQL, indexes: indexes}
		}

		snapshotDatabaseVersion, err := fsutil.GetHistorySegmentDatabaseVersion(filepath.Join(sourceDir, history.CompressedFileName()))
		if err != nil {
			return LoadResult{}, fmt.Errorf("failed to get snapshot database version: %w", err)
		}

		if dbMetaData.DatabaseVersion != snapshotDatabaseVersion {
			log.Info("migrating database", logging.Int64("current database version", dbMetaData.DatabaseVersion), logging.Int64("target database version", snapshotDatabaseVersion))

			removedConstraintsAndIndexes, err = b.migrateDatabase(ctx, log, removedConstraintsAndIndexes,
				snapshotDatabaseVersion, withIndexesAndOrderTriggers)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to migrate database from version %d to %d", dbMetaData.DatabaseVersion, snapshotDatabaseVersion)
			}
			log.Infof("finished migrating database from version %d to version %d", dbMetaData.DatabaseVersion, snapshotDatabaseVersion)

			// After migration update the database meta-data
			dbMetaData, err = NewDatabaseMetaData(ctx, b.connPool)
			if err != nil {
				return LoadResult{}, fmt.Errorf("failed to get database meta data after database migration: %w", err)
			}

		}

		rowsCopied, err = b.loadSnapshot(ctx, log, history, sourceDir, dbMetaData, b.connPool, withIndexesAndOrderTriggers)
		if err != nil {
			return LoadResult{}, fmt.Errorf("failed to load history snapshot %s: %w", history, err)
		}
		totalRowsCopied += rowsCopied
	}

	// Then current state

	if err = b.truncateCurrentStateTables(ctx, dbMetaData); err != nil {
		return LoadResult{}, fmt.Errorf("failed to truncate current state tables: %w", err)
	}

	rowsCopied, err = b.loadSnapshot(ctx, log, currentStateSnapshot, sourceDir, dbMetaData, b.connPool, withIndexesAndOrderTriggers)
	totalRowsCopied += rowsCopied
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to load current state snapshot %s: %w", currentStateSnapshot, err)
	}

	log.Infof("creating all indexes")
	err = b.createAllIndexes(ctx, log, b.connPool, removedConstraintsAndIndexes.indexes)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Infof("restoring all constraints")
	err = b.restoreAllConstraints(ctx, removedConstraintsAndIndexes.createConstraintsSQL)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to restore all constraints: %w", err)
	}

	log.Infof("recreating continuous aggregate data")
	err = b.recreateAllContinuousAggregateData(ctx, b.connPool)
	if err != nil {
		return LoadResult{}, fmt.Errorf("failed to recreate continuous aggregate data: %w", err)
	}

	if !withIndexesAndOrderTriggers {
		log.Infof("restoring current orders set")
		err = orders.RestoreCurrentOrdersSet(ctx, b.connPool)
		if err != nil {
			return LoadResult{}, fmt.Errorf("failed to restore current orders set: %w", err)
		}
	}
	return LoadResult{
		LoadedFromHeight: heightToLoadFrom,
		LoadedToHeight:   historyToHeight,
		RowsLoaded:       totalRowsCopied,
	}, nil
}

func (b *Service) migrateDatabase(ctx context.Context, log LoadLog, constraintsSqlAndIndexes *constraintsAndIndexes,
	snapshotDatabaseVersion int64, withIndexesAndOrderTriggers bool,
) (*constraintsAndIndexes, error) {
	log.Infof("restoring all indexes prior to schema migration")
	err := b.createAllIndexes(ctx, log, b.connPool, constraintsSqlAndIndexes.indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Infof("restoring all constraints prior to schema migration")
	err = b.restoreAllConstraints(ctx, constraintsSqlAndIndexes.createConstraintsSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to restore all constraints: %w", err)
	}

	log.Infof("migrating schema to version %d", snapshotDatabaseVersion)
	err = b.migrateSchemaToVersion(snapshotDatabaseVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate schema to version %d: %w", snapshotDatabaseVersion, err)
	}

	log.Infof("removing constraints after schema migration")
	constraintsSQL, err := b.removeConstraints(ctx, log, b.connPool)
	if err != nil {
		return nil, fmt.Errorf("failed to remove constraints: %w", err)
	}

	log.Infof("removing indexes after schema migration")
	indexes, err := b.dropAllIndexes(ctx, log, b.connPool, withIndexesAndOrderTriggers)
	if err != nil {
		return nil, fmt.Errorf("failed to drop all indexes: %w", err)
	}

	return &constraintsAndIndexes{createConstraintsSQL: constraintsSQL, indexes: indexes}, nil
}

func (b *Service) restoreAllConstraints(ctx context.Context, constraintsSQL []string) error {
	for _, constraintSQL := range constraintsSQL {
		_, err := b.connPool.Exec(ctx, constraintSQL)
		if err != nil {
			return fmt.Errorf("failed to execute create constraint sql %s: %w", constraintSQL, err)
		}
	}
	return nil
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

func (b *Service) truncateCurrentStateTables(ctx context.Context, dbMetaData DatabaseMetadata) error {
	for tableName := range dbMetaData.TableNameToMetaData {
		if !dbMetaData.TableNameToMetaData[tableName].Hypertable {
			tableTruncateSQL := fmt.Sprintf("truncate table %s", tableName)
			_, err := b.connPool.Exec(ctx, tableTruncateSQL)
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

func (b *Service) loadSnapshot(ctx context.Context, loadLog LoadLog, snapshotData compressedFileMapping, copyFromDirectory string,
	dbMetaData DatabaseMetadata, vegaDbConn *pgxpool.Pool, withIndexesAndOrderTriggers bool,
) (int64, error) {
	compressedFilePath := filepath.Join(copyFromDirectory, snapshotData.CompressedFileName())
	decompressedFilesDestination := filepath.Join(copyFromDirectory, snapshotData.UncompressedDataDir())
	defer func() {
		_ = os.RemoveAll(compressedFilePath)
		_ = os.RemoveAll(decompressedFilesDestination)
	}()

	loadLog.Infof("decompressing %s", snapshotData.CompressedFileName())
	err := fsutil.DecompressAndUntarFile(compressedFilePath, decompressedFilesDestination)
	if err != nil {
		return 0, fmt.Errorf("failed to decompress and untar data: %w", err)
	}

	loadLog.Infof("copying %s into database", snapshotData.UncompressedDataDir())

	rowsCopied, err := b.copyDataIntoDatabase(ctx, vegaDbConn, decompressedFilesDestination,
		filepath.Join(b.snapshotsCopyFromPath, snapshotData.UncompressedDataDir()), dbMetaData,
		withIndexesAndOrderTriggers)
	if err != nil {
		return 0, fmt.Errorf("failed to copy uncompressed data into the database %s : %w", snapshotData.UncompressedDataDir(), err)
	}

	loadLog.Infof("copied %d rows from %s into database", rowsCopied, snapshotData.UncompressedDataDir())

	return rowsCopied, nil
}

func getLastPartitionColumnEntryForHistoryTable(ctx context.Context, vegaDbConn *pgxpool.Conn, historyTableMetaData TableMetadata) (time.Time, error) {
	timeSelect := fmt.Sprintf(`SELECT %s FROM %s order by %s desc limit 1`, historyTableMetaData.PartitionColumn, historyTableMetaData.Name,
		historyTableMetaData.PartitionColumn)

	rows, err := vegaDbConn.Query(ctx, timeSelect)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query last partition column time: %w", err)
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

type constraintsAndIndexes struct {
	indexes              []IndexInfo
	createConstraintsSQL []string
}

func (b *Service) removeConstraints(ctx context.Context, loadLog LoadLog, vegaDbConn Conn) ([]string, error) {
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

	loadLog.Infof("dropping all constraints")
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

	return createConstraintsSQL, nil
}

func (b *Service) dropAllIndexes(ctx context.Context, loadLog LoadLog, vegaDbConn Conn, withIndexesAndOrderTriggers bool) ([]IndexInfo, error) {
	var indexes []IndexInfo
	if !withIndexesAndOrderTriggers {
		err := pgxscan.Select(ctx, vegaDbConn, &indexes,
			`select tablename, Indexname, Indexdef from pg_indexes where schemaname ='public' order by tablename`)
		if err != nil {
			return nil, fmt.Errorf("failed to get table indexes: %w", err)
		}

		loadLog.Infof("dropping all indexes")
		for _, index := range indexes {
			_, err = vegaDbConn.Exec(ctx, fmt.Sprintf("DROP INDEX %s", index.Indexname))
			if err != nil {
				return nil, fmt.Errorf("failed to drop index %s: %w", index.Indexname, err)
			}
		}
	}
	return indexes, nil
}

func (b *Service) copyDataIntoDatabase(ctx context.Context, pool *pgxpool.Pool, copyFromDir string,
	databaseCopyFromDir string, dbMetaData DatabaseMetadata,
	withIndexesAndOrderTriggers bool,
) (int64, error) {
	files, err := os.ReadDir(copyFromDir)
	if err != nil {
		return 0, fmt.Errorf("failed to get files in snapshot dir: %w", err)
	}

	// Disable all triggers
	vegaDbConn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer vegaDbConn.Release()

	_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = replica;")
	if err != nil {
		return 0, fmt.Errorf("failed to disable triggers, setting session replication role to replica failed: %w", err)
	}
	defer func() {
		_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = DEFAULT;")
		if err != nil {
			b.log.Errorf("failed to enable triggers, setting session replication role to DEFAULT failed: %w", err)
		}
	}()

	var totalRowsCopied int64
	for _, file := range files {
		if !file.IsDir() {
			tableName := file.Name()
			if sqlstore.OrdersTableName == tableName && withIndexesAndOrderTriggers {
				_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = DEFAULT;")
				if err != nil {
					b.log.Errorf("failed to enable triggers prior to copying data into orders table, setting session replication role to DEFAULT failed: %w", err)
				}
			}
			rowsCopied, err := b.copyTableDataIntoDatabase(ctx, dbMetaData.TableNameToMetaData[tableName], databaseCopyFromDir, vegaDbConn)
			if err != nil {
				return 0, fmt.Errorf("failed to copy data into table %s: %w", tableName, err)
			}

			if sqlstore.OrdersTableName == tableName && withIndexesAndOrderTriggers {
				_, err = vegaDbConn.Exec(ctx, "SET session_replication_role = replica;")
				if err != nil {
					return 0, fmt.Errorf("failed to disable triggers after copying data into orders table, setting session replication role to replica failed: %w", err)
				}
			}

			totalRowsCopied += rowsCopied
		}
	}

	return totalRowsCopied, nil
}

func (b *Service) copyTableDataIntoDatabase(ctx context.Context, tableMetaData TableMetadata, databaseCopyFromDir string,
	vegaDbConn *pgxpool.Conn,
) (int64, error) {
	var err error
	var rowsCopied int64

	snapshotFilePath := filepath.Join(databaseCopyFromDir, tableMetaData.Name)
	if tableMetaData.Hypertable {
		rowsCopied, err = b.copyHistoryTableDataIntoDatabase(ctx, tableMetaData, snapshotFilePath, vegaDbConn)
		if err != nil {
			return 0, fmt.Errorf("failed to copy history table data into database: %w", err)
		}
	} else {
		rowsCopied, err = b.copyCurrentStateTableDataIntoDatabase(ctx, tableMetaData, vegaDbConn, snapshotFilePath)
		if err != nil {
			return 0, fmt.Errorf("failed to copy current state table data into database: %w", err)
		}
	}
	return rowsCopied, nil
}

func (b *Service) copyCurrentStateTableDataIntoDatabase(ctx context.Context, tableMetaData TableMetadata, vegaDbConn *pgxpool.Conn, snapshotFilePath string) (int64, error) {
	tag, err := vegaDbConn.Exec(ctx, fmt.Sprintf(`copy %s from '%s'`, tableMetaData.Name, snapshotFilePath))
	if err != nil {
		return 0, fmt.Errorf("failed to copy data into current state table: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (b *Service) copyHistoryTableDataIntoDatabase(ctx context.Context, tableMetaData TableMetadata, snapshotFilePath string, vegaDbConn *pgxpool.Conn) (int64, error) {
	lastPartitionColumnEntry, err := getLastPartitionColumnEntryForHistoryTable(ctx, vegaDbConn, tableMetaData)
	if err != nil {
		return 0, fmt.Errorf("failed to get last partition column entry for table %s: %w", tableMetaData.Name, err)
	}

	partitionColumn := tableMetaData.PartitionColumn
	timestampString, err := encodeTimestampToString(lastPartitionColumnEntry)
	if err != nil {
		return 0, fmt.Errorf("failed to encode timestamp into string: %w", err)
	}

	copyQuery := fmt.Sprintf(`copy %s from '%s' where %s > timestamp '%s'`, tableMetaData.Name, snapshotFilePath,
		partitionColumn, timestampString)

	tag, err := vegaDbConn.Exec(ctx, copyQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to copy data into hyper-table: %w", err)
	}

	return tag.RowsAffected(), nil
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

func (b *Service) createAllIndexes(ctx context.Context, loadLog LoadLog, vegaDbConn Conn, indexes []IndexInfo) error {
	for _, index := range indexes {
		loadLog.Infof("creating index %s", index.Indexname)
		_, err := vegaDbConn.Exec(ctx, index.Indexdef)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", index.Indexname, err)
		}
	}
	return nil
}

func (b *Service) recreateAllContinuousAggregateData(ctx context.Context, vegaDbConn Conn) error {
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
