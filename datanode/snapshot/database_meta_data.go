package snapshot

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
)

type DatabaseMetadata struct {
	TableNameToMetaData map[string]TableMetadata
	DatabaseVersion     int64
}

type TableMetadata struct {
	Name       string
	SortOrder  string
	Hypertable bool
}

type IndexInfo struct {
	Tablename string
	Indexname string
	Indexdef  string
}

func NewDatabaseMetaData(ctx context.Context, connConfig sqlstore.ConnectionConfig) (DatabaseMetadata, error) {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	tableNames, err := GetAllTableNames(ctx, conn)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get names of tables to copy:%w", err)
	}

	tableNameToSortOrder, err := getTableSortOrders(ctx, conn)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get table sort orders:%w", err)
	}

	hyperTableNames, err := getHyperTableNames(ctx, conn)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get hyper table names:%w", err)
	}

	dbVersion, err := getDatabaseVersion(connConfig)
	if err != nil {
		return DatabaseMetadata{}, fmt.Errorf("failed to get database version:%w", err)
	}

	result := DatabaseMetadata{TableNameToMetaData: map[string]TableMetadata{}, DatabaseVersion: dbVersion}
	for _, tableName := range tableNames {
		result.TableNameToMetaData[tableName] = TableMetadata{
			Name:       tableName,
			SortOrder:  tableNameToSortOrder[tableName],
			Hypertable: hyperTableNames[tableName],
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

func getDatabaseVersion(connConfig sqlstore.ConnectionConfig) (int64, error) {
	poolConfig, err := connConfig.GetPoolConfig()
	if err != nil {
		return 0, fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	dbVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return 0, fmt.Errorf("failed to get goose database version:%w", err)
	}
	return dbVersion, nil
}

func GetAllTableNames(ctx context.Context, conn *pgxpool.Pool) ([]string, error) {
	tableNameRows, err := conn.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' and table_type = 'BASE TABLE' and table_name != 'goose_db_version' order by table_name")
	if err != nil {
		return nil, fmt.Errorf("failed to query table names:%w", err)
	}

	var tableNames []string
	for tableNameRows.Next() {
		tableName := ""
		err = tableNameRows.Scan(&tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table Name:%w", err)
		}
		tableNames = append(tableNames, tableName)
	}
	return tableNames, nil
}

func HasVegaSchema(ctx context.Context, conf sqlstore.ConnectionConfig) (bool, error) {
	conn, err := pgxpool.Connect(ctx, conf.GetConnectionString())
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	tableNames, err := GetAllTableNames(ctx, conn)
	if err != nil {
		return false, fmt.Errorf("failed to get all table names:%w", err)
	}

	return len(tableNames) != 0, nil
}

func getTableSortOrders(ctx context.Context, conn *pgxpool.Pool) (map[string]string, error) {
	var primaryKeyIndexes []IndexInfo
	err := pgxscan.Select(ctx, conn, &primaryKeyIndexes,
		`select tablename, Indexname, Indexdef from pg_indexes where schemaname ='public' and Indexname like '%_pkey' order by tablename`)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary key indexes:%w", err)
	}

	tableNameToSortOrder := map[string]string{}
	for _, pkIdx := range primaryKeyIndexes {
		split := strings.Split(pkIdx.Indexdef, "(")
		if len(split) != 2 {
			return nil, fmt.Errorf("unexpected primary key index definition:%s", pkIdx.Indexdef)
		}
		tableNameToSortOrder[pkIdx.Tablename] = strings.Replace(split[1], ")", "", 1)
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
