package sqlstore

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
	"github.com/jackc/pgtype"
	shopspring "github.com/jackc/pgtype/ext/shopspring-numeric"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	ErrBadID   = errors.New("Bad ID (must be hex string)")
	tableNames = [...]string{"ledger", "accounts", "parties", "assets", "blocks"}
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type SqlStore struct {
	conf Config
	pool *pgxpool.Pool
	log  *logging.Logger
}

func (s *SqlStore) makeConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		s.conf.Username,
		s.conf.Password,
		s.conf.Host,
		s.conf.Port,
		s.conf.Database)
}

func (s *SqlStore) makePoolConfig() (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(s.makeConnectionString())
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = "Vega Data Node"
	return cfg, nil
}

func (s *SqlStore) migrateToLatestSchema() error {
	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(s.log.Named("db migration").GooseLogger())

	db := stdlib.OpenDB(*s.pool.Config().ConnConfig)

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return err
	}

	if currentVersion > 0 && s.conf.WipeOnStartup {
		if err := goose.Down(db, "migrations"); err != nil {
			return fmt.Errorf("error clearing sql schema: %w", err)
		}
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	return nil
}

func registerNumericType(poolConfig *pgxpool.Config) {
	// Cause postgres numeric types to be loaded as shopspring decimals and vice-versa
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: &shopspring.Numeric{},
			Name:  "numeric",
			OID:   pgtype.NumericOID,
		})
		return nil
	}
}
func InitialiseStorage(log *logging.Logger, config Config) (*SqlStore, error) {
	s := SqlStore{
		conf: config,
		log:  log.Named("sql_store")}

	poolConfig, err := s.makePoolConfig()
	if err != nil {
		return nil, fmt.Errorf("error configuring database: %w", err)
	}

	registerNumericType(poolConfig)

	if s.pool, err = pgxpool.ConnectConfig(context.Background(), poolConfig); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	if err = s.migrateToLatestSchema(); err != nil {
		return nil, fmt.Errorf("error migrating schema: %w", err)
	}

	return &s, nil
}

func (s *SqlStore) DeleteEverything() error {
	for _, table := range tableNames {
		if _, err := s.pool.Exec(context.Background(), "truncate table "+table+" CASCADE"); err != nil {
			return fmt.Errorf("error truncating table: %s %w", table, err)
		}
	}
	return nil
}
