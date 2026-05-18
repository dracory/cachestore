package cachestore

import (
	"context"
	"database/sql"
)

type StoreInterface interface {
	// DriverName returns the name of the database driver
	DriverName(db *sql.DB) string

	// EnableDebug enables or disables debug mode
	EnableDebug(debugEnabled bool)

	// GetCacheTableName returns the cache table name
	GetCacheTableName() string

	// SetCacheTableName sets the cache table name
	SetCacheTableName(cacheTableName string)

	// MigrateDown drops the cache table
	MigrateDown(tx *sql.Tx) error

	// MigrateUp creates the cache table
	MigrateUp(tx *sql.Tx) error

	// ExpireCacheGoroutine runs the cache expiration goroutine
	ExpireCacheGoroutine(ctx context.Context) error

	// Set stores a value in the cache
	Set(key string, value string, seconds int64) error

	// Get retrieves a value from the cache
	Get(key string, valueDefault string) (string, error)

	// SetJSON stores a JSON value in the cache
	SetJSON(key string, value any, seconds int64) error

	// GetJSON retrieves a JSON value from the cache
	GetJSON(key string, valueDefault any) (any, error)

	// Remove removes a value from the cache
	Remove(key string) error

	// FindByKey finds a cache entry by key
	FindByKey(key string) (*Cache, error)
}
