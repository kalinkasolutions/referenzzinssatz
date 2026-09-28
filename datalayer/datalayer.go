package datalayer

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kalinkasolutions/referenzzinssatz/config"
	_ "github.com/mattn/go-sqlite3"
	"log/slog"
)

// Each entry upgrades the schema by one version, tracked in PRAGMA user_version.
// Never edit an entry once released; append a new one instead.
var migrations = []string{
	// Databases created before versioning already have these tables.
	`CREATE TABLE IF NOT EXISTS Subscribers (
		Id				TEXT PRIMARY KEY NOT NULL,
		CreatedAt		TEXT,
		Email			TEXT,
		EmailValidated	INTEGER,
		ValidationCode	TEXT,
		UnsubscribeCode TEXT
	);
	CREATE TABLE IF NOT EXISTS InterestRates (
		Id							TEXT PRIMARY KEY NOT NULL,
		CreatedAt					TEXT,
		ReferenceInterestRate		REAL,
		ValidFrom					TEXT,
		UnderlyingAvgInterestRate	REAL,
		ReferenceDateOfSurvey		TEXT
	);
	CREATE TABLE IF NOT EXISTS Logs (
		Id							TEXT PRIMARY KEY NOT NULL,
		CreatedAt					TEXT,
		LogLevel					INTEGER,
		Message						TEXT
	);`,

	// Dates used to be stored as scraped (dd.mm.yyyy); ISO dates sort correctly.
	`UPDATE InterestRates SET ValidFrom = substr(trim(ValidFrom), 7, 4) || '-' || substr(trim(ValidFrom), 4, 2) || '-' || substr(trim(ValidFrom), 1, 2)
		WHERE trim(ValidFrom) GLOB '[0-9][0-9].[0-9][0-9].[0-9][0-9][0-9][0-9]';
	UPDATE InterestRates SET ReferenceDateOfSurvey = substr(trim(ReferenceDateOfSurvey), 7, 4) || '-' || substr(trim(ReferenceDateOfSurvey), 4, 2) || '-' || substr(trim(ReferenceDateOfSurvey), 1, 2)
		WHERE trim(ReferenceDateOfSurvey) GLOB '[0-9][0-9].[0-9][0-9].[0-9][0-9][0-9][0-9]';
	DELETE FROM InterestRates WHERE rowid NOT IN (
		SELECT min(rowid) FROM InterestRates
		GROUP BY ReferenceInterestRate, ValidFrom, UnderlyingAvgInterestRate, ReferenceDateOfSurvey
	);`,

	// Keep one row per address, preferring a confirmed one, so the unique index can be created.
	`UPDATE Subscribers SET Email = lower(trim(Email));
	DELETE FROM Subscribers WHERE Id NOT IN (
		SELECT Id FROM (
			SELECT Id, ROW_NUMBER() OVER (PARTITION BY Email ORDER BY EmailValidated DESC, CreatedAt) AS position
			FROM Subscribers
		) WHERE position = 1
	);
	CREATE UNIQUE INDEX IF NOT EXISTS SubscribersEmail ON Subscribers (Email);`,

	// Logs come from slog now: its level numbers, plus the record's fields as JSON.
	`ALTER TABLE Logs ADD COLUMN Attributes TEXT NOT NULL DEFAULT '';
	UPDATE Logs SET LogLevel = CASE LogLevel WHEN 0 THEN -4 WHEN 1 THEN 0 WHEN 2 THEN 4 WHEN 3 THEN 8 ELSE LogLevel END;`,
}

func NewDb(logger *slog.Logger, config config.Config) *sql.DB {
	path := filepath.Join(config.DatabasePath, config.DatabaseName)
	logger.Info("Initializing database", "path", path)

	if config.DatabasePath != "" {
		err := os.MkdirAll(config.DatabasePath, os.ModePerm)
		if err != nil {
			logger.Error("Failed to create database directory", "path", config.DatabasePath, "error", err)
			os.Exit(1)
		}
	}

	db, err := sql.Open("sqlite3", withConnectionOptions(path))
	if err != nil {
		logger.Error("Failed to open database", "path", path, "error", err)
		os.Exit(1)
	}

	migrate(logger, db)
	return db
}

// The scheduler and HTTP handlers write concurrently; without a busy timeout
// SQLite fails those writes with "database is locked" instead of waiting.
func withConnectionOptions(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "_busy_timeout=5000&_journal_mode=WAL"
}

func migrate(logger *slog.Logger, db *sql.DB) {
	version := getCurrentVersion(logger, db)
	logger.Info("Database schema", "version", version, "latest", len(migrations))

	for ; version < len(migrations); version++ {
		if err := applyMigration(db, migrations[version], version+1); err != nil {
			logger.Error("Failed to migrate database", "version", version+1, "error", err)
			os.Exit(1)
		}
	}
}

func applyMigration(db *sql.DB, migration string, targetVersion int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migration); err != nil {
		return err
	}
	// PRAGMA does not accept bound parameters.
	if _, err := tx.Exec("PRAGMA user_version = " + strconv.Itoa(targetVersion)); err != nil {
		return err
	}
	return tx.Commit()
}

func getCurrentVersion(logger *slog.Logger, db *sql.DB) int {
	var version int
	err := db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		logger.Error("Failed to get database version", "error", err)
		os.Exit(1)
	}
	return version
}
