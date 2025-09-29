package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"nebula_manager/internal/config"
	"nebula_manager/internal/utils"
)

var db *gorm.DB

// Connect initialises the global database connection using configuration settings.
func Connect(cfg *config.Config) *gorm.DB {
	if db != nil {
		return db
	}

	dsn := cfg.DatabaseDSN
	if err := ensureDatabase(cfg); err != nil {
		log.Fatalf("failed to prepare database: %v", err)
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	conn, err := gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := conn.DB()
	if err != nil {
		log.Fatalf("failed to get database handle: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	db = conn
	return db
}

func ensureDatabase(cfg *config.Config) error {
	info, err := utils.ParseMySQLDSN(cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	if info.DBName == "" {
		return fmt.Errorf("database name missing in DSN")
	}

	noDBDSN := utils.BuildMySQLDSN(info, "")
	conn, err := sql.Open("mysql", noDBDSN)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", info.DBName)
	if _, err := conn.Exec(query); err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	return nil
}

// DB exposes the initialised database object. Panic if not connected.
func DB() *gorm.DB {
	if db == nil {
		log.Fatal("database connection has not been initialised")
	}
	return db
}
