// Copyright (c) 2022 Gobalsky Labs Limited

//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"time"

	"go.uber.org/zap"

	"code.vegaprotocol.io/vega/datanode/entities"

	"github.com/jackc/pgx/v4/pgxpool"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"github.com/shopspring/decimal"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

var ErrBadID = errors.New("bad id (must be hex string)")

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

const (
	SQLMigrationsDir = "migrations"
	InfiniteInterval = "forever"
	blocksEntity     = "blocks"
)

var defaultRetentionPolicies = map[RetentionPeriod][]RetentionPolicy{
	RetentionPeriodStandard: {
		{HypertableOrCaggName: "balances", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "checkpoints", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "conflated_balances", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "delegations", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "ledger", DataRetentionPeriod: "6 months"},
		{HypertableOrCaggName: "orders", DataRetentionPeriod: "1 month"},
		{HypertableOrCaggName: "trades", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "trades_candle_1_minute", DataRetentionPeriod: "1 month"},
		{HypertableOrCaggName: "trades_candle_5_minutes", DataRetentionPeriod: "1 month"},
		{HypertableOrCaggName: "trades_candle_15_minutes", DataRetentionPeriod: "1 month"},
		{HypertableOrCaggName: "trades_candle_1_hour", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "trades_candle_6_hours", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "trades_candle_1_day", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "market_data", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "margin_levels", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "conflated_margin_levels", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "positions", DataRetentionPeriod: "7 days"},
		{HypertableOrCaggName: "conflated_positions", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "liquidity_provisions", DataRetentionPeriod: "1 day"},
		{HypertableOrCaggName: "markets", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "deposits", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "withdrawals", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "blocks", DataRetentionPeriod: "1 year"},
		{HypertableOrCaggName: "rewards", DataRetentionPeriod: "1 year"},
	},
	RetentionPeriodArchive: {
		{HypertableOrCaggName: "*", DataRetentionPeriod: string(RetentionPeriodArchive)},
	},
	RetentionPeriodLite: {
		{HypertableOrCaggName: "*", DataRetentionPeriod: string(RetentionPeriodLite)},
	},
}

func MigrateToLatestSchema(log *logging.Logger, config Config) error {
	log = log.Named("db-migrate")
	goose.SetBaseFS(EmbedMigrations)
	goose.SetLogger(log.GooseLogger())
	goose.SetVerbose(bool(config.VerboseMigration))

	poolConfig, err := config.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	log.Info("Checking database version and migrating sql schema to latest version, please wait...")
	if err = goose.Up(db, SQLMigrationsDir); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	log.Info("Sql schema migration completed successfully")

	return nil
}

func MigrateToSchemaVersion(log *logging.Logger, config Config, version int64, fs fs.FS) error {
	goose.SetBaseFS(fs)
	goose.SetLogger(log.Named("db migration").GooseLogger())
	goose.SetVerbose(bool(config.VerboseMigration))
	goose.SetVerbose(true)

	poolConfig, err := config.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	log.Infof("Checking database version and migrating sql schema to version %d, please wait...", version)
	if err = goose.UpTo(db, SQLMigrationsDir, version); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	log.Info("Sql schema migration completed successfully")

	return nil
}

func RevertToSchemaVersionZero(log *logging.Logger, config ConnectionConfig, fs fs.FS, verbose bool) error {
	log = log.Named("revert-schema-to-version-0")
	goose.SetBaseFS(fs)
	goose.SetLogger(log.GooseLogger())
	goose.SetVerbose(verbose)

	poolConfig, err := config.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	log.Info("Checking database version and reverting sql schema to version 0, please wait...")
	if err := goose.DownTo(db, SQLMigrationsDir, 0); err != nil {
		return fmt.Errorf("failed to goose down the schema to version 0: %w", err)
	}
	log.Info("Sql schema migration completed successfully")

	return nil
}

func WipeDatabaseAndMigrateSchemaToVersion(log *logging.Logger, config ConnectionConfig, version int64, fs fs.FS, verbose bool) error {
	log = log.Named("db-wipe-migrate")
	goose.SetBaseFS(fs)
	goose.SetLogger(log.GooseLogger())
	goose.SetVerbose(verbose)

	poolConfig, err := config.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return err
	}

	log.Infof("Wiping database and migrating schema to version %d", version)
	if currentVersion > 0 {
		if err := goose.DownTo(db, SQLMigrationsDir, 0); err != nil {
			return fmt.Errorf("failed to goose down the schema: %w", err)
		}
	}

	if version > 0 {
		if err := goose.UpTo(db, SQLMigrationsDir, version); err != nil {
			return fmt.Errorf("failed to goose up the schema: %w", err)
		}
	}
	log.Info("Sql schema migration completed successfully")

	return nil
}

func WipeDatabaseAndMigrateSchemaToLatestVersion(log *logging.Logger, config ConnectionConfig, fs fs.FS, verbose bool) error {
	log = log.Named("db-wipe-migrate")
	goose.SetBaseFS(fs)
	goose.SetLogger(log.GooseLogger())
	goose.SetVerbose(verbose)

	poolConfig, err := config.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get pool config:%w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return err
	}

	log.Info("Wiping database and migrating schema to latest version")
	if currentVersion > 0 {
		if err := goose.DownTo(db, SQLMigrationsDir, 0); err != nil {
			return fmt.Errorf("failed to goose down the schema: %w", err)
		}
	}

	if err := goose.Up(db, SQLMigrationsDir); err != nil {
		return fmt.Errorf("failed to goose up the schema: %w", err)
	}
	log.Info("Sql schema migration completed successfully")

	return nil
}

func HasVegaSchema(ctx context.Context, conn Connection) (bool, error) {
	tableNames, err := GetAllTableNames(ctx, conn)
	if err != nil {
		return false, fmt.Errorf("failed to get all table names:%w", err)
	}

	return len(tableNames) != 0, nil
}

func GetAllTableNames(ctx context.Context, conn Connection) ([]string, error) {
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

func RecreateVegaDatabase(ctx context.Context, log *logging.Logger, connConfig ConnectionConfig) error {
	postgresDbConn, err := pgx.Connect(context.Background(), connConfig.GetConnectionStringForPostgresDatabase())
	if err != nil {
		return fmt.Errorf("unable to connect to database:%w", err)
	}

	defer func() {
		err := postgresDbConn.Close(ctx)
		if err != nil {
			log.Errorf("error closing database connection after loading snapshot:%v", err)
		}
	}()

	err = dropDatabaseWithRetry(ctx, postgresDbConn, connConfig)
	if err != nil {
		return fmt.Errorf("failed to drop database:%w", err)
	}

	_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE template0 LC_COLLATE 'C' LC_CTYPE 'C';", connConfig.Database))
	if err != nil {
		return fmt.Errorf("unable to create database:%w", err)
	}
	return nil
}

type DatanodeBlockSpan struct {
	FromHeight int64
	ToHeight   int64
	HasData    bool
}

func GetDatanodeBlockSpan(ctx context.Context, connPool *pgxpool.Pool) (DatanodeBlockSpan, error) {
	hasVegaSchema, err := HasVegaSchema(ctx, connPool)
	if err != nil {
		return DatanodeBlockSpan{}, fmt.Errorf("failed to get check is database if empty:%w", err)
	}

	var span DatanodeBlockSpan
	if hasVegaSchema {
		oldestBlock, err := GetOldestHistoryBlockUsingConnection(ctx, connPool)
		if err != nil {
			if errors.Is(err, entities.ErrNotFound) {
				return DatanodeBlockSpan{
					HasData: false,
				}, nil
			}
			return DatanodeBlockSpan{}, fmt.Errorf("failed to get oldest history block:%w", err)
		}

		lastBlock, err := GetLastBlockUsingConnection(ctx, connPool)
		if err != nil {
			return DatanodeBlockSpan{}, fmt.Errorf("failed to get last block:%w", err)
		}

		span = DatanodeBlockSpan{
			FromHeight: oldestBlock.Height,
			ToHeight:   lastBlock.Height,
			HasData:    true,
		}
	}

	return span, nil
}

func dropDatabaseWithRetry(parentCtx context.Context, postgresDbConn *pgx.Conn, connConfig ConnectionConfig) error {
	var err error
	for i := 0; i < 5; i++ {
		ctx, cancelFn := context.WithTimeout(parentCtx, 20*time.Second)
		_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH ( FORCE )", connConfig.Database))
		cancelFn()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("unable to drop existing database:%w", err)
	}
	return nil
}

const oneDayAsSeconds = 60 * 60 * 24

func getRetentionEntities(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
select view_name as table_name
from timescaledb_information.continuous_aggregates
union all
select hypertable_name
from timescaledb_information.hypertables
`)
	if err != nil {
		return nil, err
	}

	retentionEntities := make([]string, 0)
	defer rows.Close()

	for rows.Next() {
		var entity string
		err = rows.Scan(&entity)
		if err != nil {
			return nil, err
		}
		retentionEntities = append(retentionEntities, entity)
	}

	return retentionEntities, nil
}

func getPolicy(entity string, policies []RetentionPolicy) (RetentionPolicy, bool) {
	for _, override := range policies {
		if override.HypertableOrCaggName == entity {
			return override, true
		}
	}
	return RetentionPolicy{}, false
}

func setRetentionPolicy(db *sql.DB, entity string, policy string, log *logging.Logger) error {
	if policy == "" {
		return nil
	}
	if _, err := db.Exec(fmt.Sprintf("SELECT remove_retention_policy('%s', true);", entity)); err != nil {
		return fmt.Errorf("removing retention policy from %s: %w", entity, err)
	}

	log.Info("Setting retention policy", zap.String("entity", entity), zap.String("policy", policy))
	// If we're keeping data forever, don't bother adding a policy at all
	if policy == InfiniteInterval {
		return nil
	}

	if _, err := db.Exec(fmt.Sprintf("SELECT add_retention_policy('%s', INTERVAL '%s');", entity, policy)); err != nil {
		return fmt.Errorf("adding retention policy to %s: %w", entity, err)
	}

	return nil
}

func setChunkInterval(db *sql.DB, entity string, interval string, log *logging.Logger) error {
	if interval == "" {
		return nil
	}

	log.Info("Setting chunk interval", zap.String("entity", entity), zap.String("interval", interval))
	if _, err := db.Exec(fmt.Sprintf("SELECT set_chunk_time_interval('%s', INTERVAL '%s');", entity, interval)); err != nil {
		return fmt.Errorf("setting chunk interval for %s: %w", entity, err)
	}

	return nil
}

func ApplyDataRetentionPolicies(config Config, log *logging.Logger) error {
	poolConfig, err := config.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return errors.Wrap(err, "applying data retention policy")
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	// get the hypertables and caggs that have been created for data node
	retentionEntities, err := getRetentionEntities(db)
	if err != nil {
		// We should panic here because something must be wrong
		panic(fmt.Errorf("getting entities with retention policies: %w", err))
	}

	// This is the default retention period the data-node is operating with
	retentionPeriod := config.RetentionPeriod
	// These are any retention policy overrides that have been set by the user
	overridePolicies := config.RetentionPolicies

	defaultPolicies := defaultRetentionPolicies[retentionPeriod]

	var maxRetentionPeriodInSecs int64
	var blocksRetentionPolicy string

	for _, entity := range retentionEntities {
		if retentionPeriod == RetentionPeriodLite || retentionPeriod == RetentionPeriodArchive {
			policy := defaultPolicies[0]
			override, ok := getPolicy(entity, overridePolicies)
			if ok { // we have found an override policy so apply it instead of the default
				// make sure that if any part of the override policy is empty, we use the default
				if override.DataRetentionPeriod == "" {
					override.DataRetentionPeriod = policy.DataRetentionPeriod
				}
				if override.ChunkInterval == "" {
					override.ChunkInterval = policy.ChunkInterval
				}
				policy = override
			}

			// Set the default retention period
			if err := setRetentionPolicy(db, entity, policy.DataRetentionPeriod, log); err != nil {
				return fmt.Errorf("setting retention policy for %s to %s: %w", entity, policy.DataRetentionPeriod, err)
			}

			if err := setChunkInterval(db, entity, policy.ChunkInterval, log); err != nil {
				return fmt.Errorf("setting chunk interval for %s to %s: %w", entity, policy.ChunkInterval, err)
			}

			continue
		}

		if entity == blocksEntity {
			// we should ignore this for now because blocks retention policy needs to be as long as the longest retention period
			continue
		}

		// if the retention period is the standard period, we need to check that a default has been defined, otherwise we should panic
		policy, ok := getPolicy(entity, defaultPolicies)
		if !ok {
			// The development team have omitted a default retention policy for this entity, we should panic here.
			panic(fmt.Errorf("no default retention policy defined for %s", entity))
		}

		override, ok := getPolicy(entity, overridePolicies)
		if ok { // we have found an override policy so apply it instead of the default
			// make sure that if any part of the override policy is empty, we use the default
			if override.DataRetentionPeriod == "" {
				override.DataRetentionPeriod = policy.DataRetentionPeriod
			}
			if override.ChunkInterval == "" {
				override.ChunkInterval = policy.ChunkInterval
			}

			policy = override
		}

		if err := setChunkInterval(db, entity, policy.ChunkInterval, log); err != nil {
			return fmt.Errorf("setting chunk interval for %s to %s: %w", entity, policy.ChunkInterval, err)
		}

		aboveMinimum, retentionPeriodInSecs, err := checkPolicyPeriodIsAtOrAboveMinimum(oneDayAsSeconds, policy, db)
		if err != nil {
			return fmt.Errorf("checking retention policy period is above minimum:%w", err)
		}

		if retentionPeriodInSecs > maxRetentionPeriodInSecs {
			maxRetentionPeriodInSecs = retentionPeriodInSecs
			blocksRetentionPolicy = policy.DataRetentionPeriod
		}

		if !config.DisableMinRetentionPolicyCheckForUseInSysTestsOnly {
			// We have this check to avoid the datanode removing data that is required for creating data snapshots
			if !aboveMinimum {
				return fmt.Errorf("policy for %s has a retention time less than one day, one day is the minimum permitted", policy.HypertableOrCaggName)
			}
		}

		// Set the default retention period
		if err := setRetentionPolicy(db, entity, policy.DataRetentionPeriod, log); err != nil {
			return fmt.Errorf("setting retention policy for %s to %s: %w", entity, policy.DataRetentionPeriod, err)
		}
	}

	// finally if the retention period is the standard period, we need to set the blocks retention policy to the longest retention period
	if retentionPeriod == RetentionPeriodStandard {
		if err := setRetentionPolicy(db, blocksEntity, blocksRetentionPolicy, log); err != nil {
			return fmt.Errorf("setting retention policy for %s to %s: %w", blocksEntity, blocksRetentionPolicy, err)
		}
	}

	return nil
}

func retentionPeriodToSeconds(db *sql.DB, retentionPeriod string) (int64, error) {
	query := fmt.Sprintf("SELECT EXTRACT(epoch FROM INTERVAL '%s')", retentionPeriod)
	row := db.QueryRow(query)

	var seconds decimal.Decimal
	err := row.Scan(&seconds)
	if err != nil {
		return 0, fmt.Errorf("failed to get interval in seconds for retention period %s: %w", retentionPeriod, err)
	}

	return seconds.IntPart(), nil
}

func checkPolicyPeriodIsAtOrAboveMinimum(minimumInSeconds int64, policy RetentionPolicy, db *sql.DB) (bool, int64, error) {
	if policy.DataRetentionPeriod == InfiniteInterval {
		return true, 0, nil
	}

	secs, err := retentionPeriodToSeconds(db, policy.DataRetentionPeriod)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get interval in seconds for policy %s: %w", policy.HypertableOrCaggName, err)
	}

	return secs >= minimumInSeconds, secs, nil
}

type EmbeddedPostgresLog interface {
	io.Writer
}

func StartEmbeddedPostgres(log *logging.Logger, config Config, runtimeDir string, postgresLog EmbeddedPostgresLog) (*embeddedpostgres.EmbeddedPostgres, error) {
	log = log.Named("embedded-postgres")
	log.SetLevel(config.Level.Get())
	embeddedPostgresDataPath := paths.JoinStatePath(paths.StatePath(runtimeDir), "node-data")

	embeddedPostgres := createEmbeddedPostgres(runtimeDir, &embeddedPostgresDataPath,
		postgresLog, config.ConnectionConfig)

	if err := embeddedPostgres.Start(); err != nil {
		log.Errorf("error starting embedded postgres: %v", err)
		return nil, fmt.Errorf("use embedded database was true, but failed to start: %w", err)
	}

	return embeddedPostgres, nil
}

func createEmbeddedPostgres(runtimePath string, dataPath *paths.StatePath, writer io.Writer, conf ConnectionConfig) *embeddedpostgres.EmbeddedPostgres {
	dbConfig := embeddedpostgres.DefaultConfig().
		Username(conf.Username).
		Password(conf.Password).
		Database(conf.Database).
		Port(uint32(conf.Port)).
		ListenAddr(conf.Host).
		SocketDir(conf.SocketDir).
		Logger(writer)

	if len(runtimePath) != 0 {
		dbConfig = dbConfig.RuntimePath(runtimePath).BinariesPath(runtimePath)
	}

	if dataPath != nil {
		dbConfig = dbConfig.DataPath(dataPath.String())
	}

	return embeddedpostgres.NewDatabase(dbConfig)
}
