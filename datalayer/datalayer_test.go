package datalayer

import (
	"log/slog"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
)

func TestEnsureSubscribersTableExist(t *testing.T) {
	db := NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Subscribers'")

	var tableName string
	row.Scan(&tableName)

	assert.Equal(t, "Subscribers", tableName)
}

func TestEnsureInterestRatesTableExist(t *testing.T) {
	db := NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'InterestRates'")

	var tableName string
	row.Scan(&tableName)

	assert.Equal(t, "InterestRates", tableName)
}

func TestEnsureInterestLogExist(t *testing.T) {
	db := NewDb(slog.New(slog.DiscardHandler), config.Config{
		DatabasePath: "",
		DatabaseName: "file::memory:?cache=shared",
	})
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'Logs'")

	var tableName string
	row.Scan(&tableName)

	assert.Equal(t, "Logs", tableName)
}
