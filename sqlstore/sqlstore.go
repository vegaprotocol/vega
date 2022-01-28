package sqlstore

import (
	"context"
	"embed"
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
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

func InitialiseStorage(log *logging.Logger, config Config) (*SqlStore, error) {
	s := SqlStore{
		conf: config,
		log:  log.Named("sql_store")}

	poolConfig, err := s.makePoolConfig()
	if err != nil {
		return nil, err
	}

	if s.pool, err = pgxpool.ConnectConfig(context.Background(), poolConfig); err != nil {
		return nil, err
	}

	if err = s.migrateToLatestSchema(); err != nil {
		return nil, err
	}

	return &s, nil
}
