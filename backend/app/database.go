package app

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
)

// OpenDatabase connects to the configured database and sizes its pool. Run and
// the reset-password command share it so both behave the same way.
func OpenDatabase(cfg *config.Config) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)
	switch cfg.Database.Driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(cfg.Database.DSN), gormConfig())
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.Database.DSN), gormConfig())
	default:
		db, err = gorm.Open(sqlite.Open(sqliteDSN(cfg.Database.DSN)), gormConfig())
	}
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.Database.Driver == "" || cfg.Database.Driver == "sqlite" {
		// SQLite has one writer at a time; a large pool only adds contenders.
		// It is not forced to 1: libraries such as the Casbin adapter use the
		// outer handle inside a transaction, which would deadlock a single
		// connection. WAL lets readers proceed and busy_timeout makes the
		// remaining writers wait instead of failing.
		pool.SetMaxOpenConns(8)
		pool.SetMaxIdleConns(8)
	} else {
		pool.SetMaxIdleConns(10)
		pool.SetMaxOpenConns(100)
	}
	return db, nil
}

// sqliteDSN adds a busy timeout and write-ahead logging unless the operator
// already chose them. Without them a brief overlap (a second process such as
// reset-password, or a backup) fails immediately instead of waiting.
func sqliteDSN(dsn string) string {
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return dsn
	}
	lower := strings.ToLower(dsn)
	if strings.Contains(lower, "_pragma") {
		return dsn // the operator manages pragmas explicitly
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	if !strings.Contains(lower, "_busy_timeout") && !strings.Contains(lower, "_timeout") {
		dsn += fmt.Sprintf("%s_busy_timeout=5000", sep)
		sep = "&"
	}
	if !strings.Contains(lower, "_journal_mode") && !strings.Contains(lower, "_journal") {
		dsn += sep + "_journal_mode=WAL"
	}
	return dsn
}

// gormConfig translates driver errors into GORM's portable ones so handlers can
// tell a duplicate key from a failure without parsing driver text.
func gormConfig() *gorm.Config { return &gorm.Config{TranslateError: true} }
