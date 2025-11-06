package db

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	config "mbflow/internal"
	"sync"
	"time"
)

var (
	bunDB   *bun.DB
	bunOnce sync.Once
)

func initBun() {
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr(config.App().PGUri()),
		pgdriver.WithInsecure(true),
		pgdriver.WithDatabase(config.App().Database.Name),
		pgdriver.WithPassword(config.App().Database.Password),
		pgdriver.WithUser(config.App().Database.User),
		pgdriver.WithTimeout(5*time.Second),
		pgdriver.WithDialTimeout(5*time.Second),
		pgdriver.WithReadTimeout(5*time.Second),
		pgdriver.WithWriteTimeout(5*time.Second),
	))
	bunDB = bun.NewDB(sqldb, pgdialect.New())
	bunDB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(config.App().Database.Debug),
	))
	_, err := bunDB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	if err != nil {
		log.Fatal().Err(err).Msg("add extension")
	}
}

func DB() *bun.DB {
	bunOnce.Do(initBun)
	return bunDB
}

func OnShutdown() error {
	return DB().Close()
}
