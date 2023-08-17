package snapshot

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pressly/goose/v3"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/shopspring/decimal"
)

type DatabaseMetadata struct {
	TableNameToMetaData                    map[string]TableMetadata
	ContinuousAggregatesMetaData           []ContinuousAggregateMetaData
	DatabaseVersion                        int64
	CurrentStateTablesCreateConstraintsSql []string
	CurrentStateTablesDropConstraintsSql   []string
	HistoryStateTablesCreateConstraintsSql []string
	HistoryStateTablesDropConstraintsSql   []string
}

type TableMetadata struct {
	Name            string
	SortOrder       string
	Hypertable      bool
	PartitionColumn string
}

type ContinuousAggregateMetaData struct {
	ID             int
	Name           string
	BucketInterval time.Duration
}

type IndexInfo struct {
	Tablename string
	Indexname string
	Indexdef  string
}

type HypertablePartitionColumns struct {
	HypertableName string
	ColumnName     string
}

func NewDatabaseMetaData(ctx context.Context, connPool *pgxpool.Pool) (DatabaseMetadata, error) {
	// Ensure timescale extension is enabled before attempting to get metadata
	_, err := connPool.Exec(ctx, "create extension if not exists timescaledb")
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to create timescale extension: %w", err)
	}

	dbVersion, err := getDBVersion(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get database version: %w", err)
	}

	if dbVersion == 0 {
		return DatabaseMetadata{}, nil
	}

	tableNames, err := sqlstore.GetAllTableNames(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get names of tables to copy:%w", err)
	}

	tableNameToSortOrder, err := getTableSortOrders(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get table sort orders:%w", err)
	}

	hyperTableNames, err := getHyperTableNames(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get hyper table names:%w", err)
	}

	hypertablePartitionColumns, err := getHyperTablePartitionColumns(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get hyper table partition columns:%w", err)
	}

	caggsMeta, err := getContinuousAggregatesMetaData(ctx, connPool)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get continuous aggregate view names:%w", err)
	}

	currentStateCreateConstraintsSql, historyCreateConstraintsSql, err := getCreateConstraintsSql(ctx, connPool, hyperTableNames)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get create constrains sql:%w", err)
	}

	currentStateDropConstraintsSql, historyDropConstraintsSql, err := getDropConstraintsSql(ctx, connPool, hyperTableNames)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get drop constrains sql:%w", err)
	}

	result := DatabaseMetadata{
		TableNameToMetaData: map[string]TableMetadata{}, DatabaseVersion: dbVersion,
		ContinuousAggregatesMetaData:           caggsMeta,
		CurrentStateTablesCreateConstraintsSql: currentStateCreateConstraintsSql,
		CurrentStateTablesDropConstraintsSql:   currentStateDropConstraintsSql,
		HistoryStateTablesCreateConstraintsSql: historyCreateConstraintsSql,
		HistoryStateTablesDropConstraintsSql:   historyDropConstraintsSql,
	}
	for _, tableName := range tableNames {
		partitionCol := ""
		ok := false
		if hyperTableNames[tableName] {
			partitionCol, ok = hypertablePartitionColumns[tableName]
			if !ok {
				return DatabaseMetadata{}, fmt.Errorf("failed to get partition column for hyper table %s", tableName)
			}
		}

		result.TableNameToMetaData[tableName] = TableMetadata{
			Name:            tableName,
			SortOrder:       tableNameToSortOrder[tableName],
			Hypertable:      hyperTableNames[tableName],
			PartitionColumn: partitionCol,
		}
	}

	return result, nil
}

func (d DatabaseMetadata) GetHistoryTableNames() []string {
	var result []string
	for _, meta := range d.TableNameToMetaData {
		if meta.Hypertable {
			result = append(result, meta.Name)
		}
	}

	return result
}

func getTableSortOrders(ctx context.Context, conn *pgxpool.Pool) (map[string]string, error) {
	var primaryKeyIndexes []IndexInfo
	err := pgxscan.Select(ctx, conn, &primaryKeyIndexes,
		`select tablename, Indexname, Indexdef from pg_indexes where schemaname ='public' and Indexname like '%_pkey' order by tablename`)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key indexes:%w", err)
	}

	includeRegexp := regexp.MustCompile(`(?i)include(\s*)\(.*$`)
	tableNameToSortOrder := map[string]string{}
	for _, pkIdx := range primaryKeyIndexes {
		withoutInclude := includeRegexp.ReplaceAllString(pkIdx.Indexdef, "")
		split := strings.Split(withoutInclude, "(")
		if len(split) != 2 {
			return nil, fmt.Errorf("unexpected primary key index definition:%s", pkIdx.Indexdef)
		}
		so := strings.Replace(split[1], ")", "", 1)
		tableNameToSortOrder[pkIdx.Tablename] = so
	}
	return tableNameToSortOrder, nil
}

func getHyperTableNames(ctx context.Context, conn *pgxpool.Pool) (map[string]bool, error) {
	tableNameRows, err := conn.Query(ctx, "SELECT hypertable_name FROM timescaledb_information.hypertables")
	if err != nil {
		return nil, fmt.Errorf("failed to query Hypertable names:%w", err)
	}

	result := map[string]bool{}
	for tableNameRows.Next() {
		tableName := ""
		err = tableNameRows.Scan(&tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table Name:%w", err)
		}
		result[tableName] = true
	}
	return result, nil
}

func getHyperTablePartitionColumns(ctx context.Context, conn *pgxpool.Pool) (map[string]string, error) {
	var partitionColumns []HypertablePartitionColumns
	err := pgxscan.Select(ctx, conn, &partitionColumns,
		`select hypertable_name, column_name from timescaledb_information.dimensions where hypertable_schema='public' and dimension_number=1`)
	if err != nil {
		return nil, fmt.Errorf("failed to partition columns:%w", err)
	}

	tableNameToPartitionColumn := map[string]string{}
	for _, column := range partitionColumns {
		tableNameToPartitionColumn[column.HypertableName] = column.ColumnName
	}
	return tableNameToPartitionColumn, nil
}

func getContinuousAggregatesMetaData(ctx context.Context, conn *pgxpool.Pool) ([]ContinuousAggregateMetaData, error) {
	var views []struct {
		ViewName       string
		ViewDefinition string
	}
	err := pgxscan.Select(ctx, conn, &views, "SELECT view_name, view_definition FROM timescaledb_information.continuous_aggregates")
	if err != nil {
		return nil, fmt.Errorf("failed to query continuous aggregate definitions:%w", err)
	}

	metas := make([]ContinuousAggregateMetaData, 0, len(views))

	for _, view := range views {
		interval, err := extractIntervalFromViewDefinition(view.ViewDefinition)
		if err != nil {
			return nil, fmt.Errorf("failed to get interval for view %s: %w", view.ViewName, err)
		}

		intervalAsDuration, err := intervalToSeconds(ctx, conn, interval)
		if err != nil {
			return nil, fmt.Errorf("failed to convert interval to seconds duration: %w", err)
		}

		query := fmt.Sprintf(`SELECT id from _timescaledb_catalog.hypertable
    			WHERE table_name=(
        		SELECT materialization_hypertable_name FROM timescaledb_information.continuous_aggregates WHERE view_name='%s');`, view.ViewName)
		row := conn.QueryRow(ctx, query)

		var caggID int
		err = row.Scan(&caggID)
		if err != nil {
			return nil, fmt.Errorf("failed to get cagg id : %w", err)
		}

		meta := ContinuousAggregateMetaData{
			ID:             caggID,
			Name:           view.ViewName,
			BucketInterval: intervalAsDuration,
		}

		metas = append(metas, meta)
	}

	return metas, nil
}

func getCreateConstraintsSql(ctx context.Context, conn *pgxpool.Pool, hyperTableNames map[string]bool) (currentState []string,
	history []string, err error,
) {
	var constraints []struct {
		Tablename string
		Sql       string
	}

	err = pgxscan.Select(ctx, conn, &constraints,
		`SELECT relname as tablename, 'ALTER TABLE '||nspname||'.'||relname||' ADD CONSTRAINT '||conname||' '|| pg_get_constraintdef(pg_constraint.oid)||';' as sql
		FROM pg_constraint 
		INNER JOIN pg_class ON conrelid=pg_class.oid 
		INNER JOIN pg_namespace ON pg_namespace.oid=pg_class.relnamespace where pg_namespace.nspname='public'
		ORDER BY CASE WHEN contype='f' THEN 0 ELSE 1 END DESC,contype DESC,nspname DESC,relname DESC,conname DESC`)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get create constraints sql:%w", err)
	}

	for _, constraint := range constraints {
		if hyperTableNames[constraint.Tablename] {
			history = append(history, constraint.Sql)
		} else {
			currentState = append(currentState, constraint.Sql)
		}
	}

	return currentState, history, nil
}

func getDropConstraintsSql(ctx context.Context, conn *pgxpool.Pool, hyperTableNames map[string]bool) (currentState []string,
	history []string, err error,
) {
	var constraints []struct {
		Tablename string
		Sql       string
	}

	err = pgxscan.Select(ctx, conn, &constraints,
		`SELECT relname as tablename, 'ALTER TABLE '||nspname||'.'||relname||' DROP CONSTRAINT '||conname||';' as sql
		FROM pg_constraint 
		INNER JOIN pg_class ON conrelid=pg_class.oid 
		INNER JOIN pg_namespace ON pg_namespace.oid=pg_class.relnamespace where pg_namespace.nspname='public' 
		ORDER BY CASE WHEN contype='f' THEN 0 ELSE 1 END,contype,nspname,relname,conname`)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get drop constraints sql:%w", err)
	}

	for _, constraint := range constraints {
		if hyperTableNames[constraint.Tablename] {
			history = append(history, constraint.Sql)
		} else {
			currentState = append(currentState, constraint.Sql)
		}
	}

	return currentState, history, nil
}

func extractIntervalFromViewDefinition(viewDefinition string) (string, error) {
	re := regexp.MustCompile(`time_bucket\('(.*)'`)
	match := re.FindStringSubmatch(viewDefinition)
	if match == nil || len(match) != 2 {
		return "", errors.New("failed to extract interval from view definition")
	}

	return match[1], nil
}

func intervalToSeconds(ctx context.Context, conn sqlstore.Connection, interval string) (time.Duration, error) {
	query := fmt.Sprintf("SELECT EXTRACT(epoch FROM INTERVAL '%s')", interval)
	row := conn.QueryRow(ctx, query)

	var seconds decimal.Decimal
	err := row.Scan(&seconds)
	if err != nil {
		return 0, fmt.Errorf("failed to get interval in seconds %s: %w", interval, err)
	}

	return time.Duration(seconds.IntPart()) * time.Second, nil
}

// getDBVersion copied from the goose library and modified to support using a pre-allocated connection. It's worth noting
// that this method also has the side effect of creating the goose version table if it does not exist as per the original
// goose code.
func getDBVersion(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	version, err := ensureDBVersion(ctx, conn)
	if err != nil {
		return -1, err
	}

	return version, nil
}

// ensureDBVersion copied from the goose library and modified to support using a pre-allocated connection.
func ensureDBVersion(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	rows, err := dbVersionQuery(ctx, conn)
	if err != nil {
		return 0, createVersionTable(ctx, conn)
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.

	toSkip := make([]int64, 0)

	for rows.Next() {
		var row goose.MigrationRecord
		if err = rows.Scan(&row.VersionID, &row.IsApplied); err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.VersionID {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// if version has been applied we're done
		if row.IsApplied {
			return row.VersionID, nil
		}

		// latest version of migration has not been applied.
		toSkip = append(toSkip, row.VersionID)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("failed to get next row: %w", err)
	}

	return 0, goose.ErrNoNextVersion
}

// dbVersionQuery copied from the goose library and modified to support using a pre-allocated connection.
func dbVersionQuery(ctx context.Context, conn *pgxpool.Pool) (pgx.Rows, error) {
	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", goose.TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

// createVersionTable copied from the goose library and modified to support using a pre-allocated connection.
func createVersionTable(ctx context.Context, conn *pgxpool.Pool) error {
	txn, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if _, err := txn.Exec(ctx, fmt.Sprintf(`CREATE TABLE %s (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, goose.TableName())); err != nil {
		txn.Rollback(ctx)
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(ctx, fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", goose.TableName()), version, applied); err != nil {
		txn.Rollback(ctx)
		return err
	}

	return txn.Commit(ctx)
}
