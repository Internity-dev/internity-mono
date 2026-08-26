// Package postgres wires the GORM connection. Schema is owned entirely by
// golang-migrate's SQL files (apps/api/migrations) — AutoMigrate is never
// called here, GORM is used purely as a query builder over tables that
// already exist (see plan section 3.2 for why).
package postgres

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(databaseURL string, verbose bool) (*gorm.DB, error) {
	logLevel := logger.Silent
	if verbose {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}
