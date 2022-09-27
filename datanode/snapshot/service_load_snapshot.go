package snapshot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/snapshot/orders"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrNoCurrentStateSnapshotFound = errors.New("no current state snapshot found")

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (b *Service) LoadAllAvailableHistory(ctx context.Context) (int64, int64, error) {
	oldestHistoryBlock, lastBlock, err := GetOldestHistoryBlockAndLastBlock(ctx, b.connConfig, b.blockStore)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get oldest history block and last block:%w", err)
	}

	chainID, currentStateSnapshot, contiguousHistory, err := GetAllAvailableHistory(b.snapshotsPath, oldestHistoryBlock, lastBlock)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get all available history: %w", err)
	}

	if datanodeHasData(oldestHistoryBlock) {
		if len(chainID) > 0 {
			datanodeChainID, err := b.chainService.GetChainID()
			if err != nil {
				return 0, 0, fmt.Errorf("failed to get datanode chain id:%w", err)
			}

			if len(datanodeChainID) > 0 {
				if chainID != datanodeChainID {
					return 0, 0, fmt.Errorf("available history chain id %s does not match datanodes existing chain id %s", chainID, datanodeChainID)
				}
			}
		}

		b.log.Infof("snapshotting all datanode data")
		meta, err := b.createSnapshot(ctx, chainID, oldestHistoryBlock.Height, lastBlock.Height, false)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to snapshot all datanode data: %w", err)
		}

		b.log.Info("snapshotted all datanode data", logging.Int64("from height", meta.HeightFrom),
			logging.Int64("from to", meta.HeightTo))
	}

	allSnapshotData := appendCurrentStateSnapshotOntoHistory(contiguousHistory, currentStateSnapshot)

	start := time.Now()
	b.log.Infof("restoring data node from snapshot data %+q", allSnapshotData)

	b.log.Infof("creating database")
	if err = sqlstore.RecreateVegaDatabase(ctx, b.log, b.connConfig); err != nil {
		return 0, 0, fmt.Errorf("failed to create vega database: %w", err)
	}

	b.log.Infof("creating schema")
	if err = sqlstore.CreateVegaSchema(b.log, b.connConfig); err != nil {
		return 0, 0, fmt.Errorf("failed to create vega schema: %w", err)
	}

	totalRowsCopied, err := b.loadSnapshotData(ctx, allSnapshotData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load snapshot data:%w", err)
	}

	toHeight, fromHeight := GetToAndFromHeightFromHistory(currentStateSnapshot, contiguousHistory)

	b.log.Info("loaded datanode history", logging.Int64("from height", fromHeight),
		logging.Int64("from to", toHeight), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", totalRowsCopied))

	return fromHeight, toHeight, nil
}

func appendCurrentStateSnapshotOntoHistory(contiguousHistory []HistorySnapshot, currentStateSnapshot *CurrentStateSnapshot) []snapshot {
	snapshotData := make([]snapshot, 0, len(contiguousHistory)+1)
	for _, history := range contiguousHistory {
		snapshotData = append(snapshotData, history)
	}
	snapshotData = append(snapshotData, currentStateSnapshot)
	return snapshotData
}

func datanodeHasData(oldestHistoryBlock *entities.Block) bool {
	return oldestHistoryBlock != nil
}

func GetAllAvailableHistory(snapshotsPath string, oldestHistoryBlock *entities.Block, lastBlock *entities.Block) (string, *CurrentStateSnapshot, []HistorySnapshot, error) {
	currentStatesChainID, currentStateSnapshots, err := GetCurrentStateSnapshots(snapshotsPath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get current state snapshots:%w", err)
	}

	historiesChainID, historySnapshots, err := GetHistorySnapshots(snapshotsPath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get history snapshots:%w", err)
	}

	chainID, err := GetChainID(currentStatesChainID, historiesChainID)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get chain id:%w", err)
	}

	currentStateSnapshot, contiguousHistory, err := GetHistoryIncludingDatanodeState(oldestHistoryBlock, lastBlock, chainID, currentStateSnapshots, historySnapshots)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get all available history:%w", err)
	}

	if currentStateSnapshot == nil {
		return "", nil, nil, ErrNoCurrentStateSnapshotFound
	}
	return chainID, currentStateSnapshot, contiguousHistory, nil
}

func GetToAndFromHeightFromHistory(currentStateSnapshot *CurrentStateSnapshot, contiguousHistory []HistorySnapshot) (int64, int64) {
	toHeight := currentStateSnapshot.Height
	fromHeight := currentStateSnapshot.Height
	for _, history := range contiguousHistory {
		if history.HeightFrom < fromHeight {
			fromHeight = history.HeightFrom
		}

		if history.HeightTo > toHeight {
			toHeight = history.HeightTo
		}
	}
	return toHeight, fromHeight
}

func GetOldestHistoryBlockAndLastBlock(ctx context.Context, connConfig sqlstore.ConnectionConfig, blockStore *sqlstore.Blocks) (*entities.Block, *entities.Block, error) {
	hasVegaSchema, err := HasVegaSchema(ctx, connConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get check if database if empty:%w", err)
	}

	var oldestHistoryBlock *entities.Block
	var lastBlock *entities.Block
	if hasVegaSchema {
		historyBlock, err := blockStore.GetOldestHistoryBlock(ctx)
		if err != nil {
			if !errors.Is(err, sqlstore.ErrNoHistoryBlock) {
				return nil, nil, fmt.Errorf("failed to get oldest history block:%w", err)
			}
		} else {
			oldestHistoryBlock = &historyBlock
		}

		block, err := blockStore.GetLastBlock(ctx)
		if err != nil {
			if !errors.Is(err, sqlstore.ErrNoLastBlock) {
				return nil, nil, fmt.Errorf("failed to get last block:%w", err)
			}
		} else {
			lastBlock = &block
		}
	}
	return oldestHistoryBlock, lastBlock, nil
}

func GetChainID(currentStatesChainID string, historiesChainID string) (string, error) {
	chainID := ""
	if len(currentStatesChainID) != 0 && len(historiesChainID) != 0 && currentStatesChainID != historiesChainID {
		return "", fmt.Errorf("current state snapshots and history snapshots have mismatched chain ids")
	}

	if len(currentStatesChainID) > 0 {
		chainID = currentStatesChainID
	} else if len(historiesChainID) > 0 {
		chainID = historiesChainID
	}

	return chainID, nil
}

func (b *Service) killAllConnectionsToDatabase(ctx context.Context) error {
	conn, err := pgxpool.Connect(ctx, b.connConfig.GetConnectionString())
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
	AND pid <> pg_backend_pid();`, b.connConfig.Database)

	_, err = conn.Exec(ctx, killAllConnectionsQuery)
	if err != nil {
		return fmt.Errorf("failed to kill all database connection:%w", err)
	}

	return nil
}

func (b *Service) loadSnapshotData(ctx context.Context, snapshotsData []snapshot) (int64, error) {
	err := b.killAllConnectionsToDatabase(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	vegaDbConn, err := pgxpool.Connect(context.Background(), b.connConfig.GetConnectionString())
	if err != nil {
		return 0, fmt.Errorf("unable to connect to vega database:%w", err)
	}

	_, err = vegaDbConn.Exec(ctx, "SET TIME ZONE 'UTC'")
	if err != nil {
		return 0, fmt.Errorf("failed to set timezone to UTC:%w", err)
	}

	b.log.Infof("preparing for bulk load")
	indexes, createConstrainsSQL, err := b.beforeBulkLoad(ctx, vegaDbConn)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare database for bulk load: %w", err)
	}

	defer func() {
		// In the event of an error attempt to clean up all extracted data
		for _, snapshotData := range snapshotsData {
			_ = os.RemoveAll(filepath.Join(b.snapshotsPath, snapshotData.UncompressedDataDir()))
		}
	}()

	dbVersion, err := getDatabaseVersion(b.connConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to get database version:%w", err)
	}

	b.log.Info("copying data into database", logging.Int64("database version", dbVersion))
	var totalRowsCopied int64
	for _, snapshotData := range snapshotsData {
		b.log.Infof("decompressing %s", snapshotData.CompressedFileName())
		snapshotDbVersion, err := decompressAndUntarSnapshot(b.log, b.snapshotsPath, snapshotData)

		// TODO handle loading of data from older database version by upgrading database as snapshots
		// TODO are loaded, for now it is expected that all snapshots are from the same database version.
		if dbVersion != snapshotDbVersion {
			return 0, fmt.Errorf("snapshot database version %d does not match current database version %d", snapshotDbVersion, dbVersion)
		}

		if err != nil {
			return 0, fmt.Errorf("failed to decompress  data: %w", err)
		}

		b.log.Infof("copying %s into database", snapshotData.UncompressedDataDir())
		rowsCopied, err := b.copyDataIntoDatabase(ctx, vegaDbConn, snapshotData.UncompressedDataDir(), b.config.DatabaseSnapshotsPath)
		if err != nil {
			b.log.Errorf("failed to copy uncompressed data into the database %s : %w", snapshotData.UncompressedDataDir(), err)
		}
		totalRowsCopied += rowsCopied
		b.log.Infof("copied %d rows from %s into database", rowsCopied, snapshotData.UncompressedDataDir())

		b.log.Infof("removing decompressed snapshot data %s", snapshotData.UncompressedDataDir())
		err = os.RemoveAll(filepath.Join(b.snapshotsPath, snapshotData.UncompressedDataDir()))
		if err != nil {
			b.log.Errorf("failed to remove decompressed data %s: %w", snapshotData.UncompressedDataDir(), err)
		}
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

func (b *Service) copyDataIntoDatabase(ctx context.Context, vegaDbConn *pgxpool.Pool, dataDir string,
	databaseSnapshotsPath string,
) (int64, error) {
	files, err := os.ReadDir(filepath.Join(b.snapshotsPath, dataDir))
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
			snapshotFilePath := filepath.Join(databaseSnapshotsPath, dataDir, file.Name())
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
