package cachestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/dracory/neat"
	contractsschema "github.com/dracory/neat/contracts/database/schema"
	"github.com/dracory/neat/database/schema/constants"
	"github.com/dracory/neat/database/soft_delete"
	neatuid "github.com/dracory/neat/support/uid"
	"github.com/dromara/carbon/v2"
)

// StoreInterface defines the interface for a cache store
type StoreInterface interface {
	// EnableDebug enables or disables debug mode
	EnableDebug(debugEnabled bool)

	// GetCacheTableName returns the cache table name
	GetCacheTableName() string

	// SetCacheTableName sets the cache table name
	SetCacheTableName(cacheTableName string)

	// MigrateDown drops the cache table
	MigrateDown(ctx context.Context, tx ...*sql.Tx) error

	// MigrateUp creates the cache table
	MigrateUp(ctx context.Context, tx ...*sql.Tx) error

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

// storeImplementation defines a cache store
type storeImplementation struct {
	db                 *neat.Database
	cacheTableName     string
	automigrateEnabled bool
	debugEnabled       bool
	logger             *slog.Logger
}

// NewStoreOptions define the options for creating a new cache store
type NewStoreOptions struct {
	DB                 *sql.DB
	CacheTableName     string
	AutomigrateEnabled bool
	DebugEnabled       bool
}

// NewStore creates a new cache store
func NewStore(opts NewStoreOptions) (StoreInterface, error) {
	if opts.DB == nil {
		return nil, errors.New("cache store: DB is required")
	}

	if opts.CacheTableName == "" {
		return nil, errors.New("cache store: CacheTableName is required")
	}

	neatDB, err := neat.NewFromSQLDB(opts.DB)
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := &storeImplementation{
		logger:             logger,
		db:                 neatDB,
		cacheTableName:     opts.CacheTableName,
		automigrateEnabled: opts.AutomigrateEnabled,
		debugEnabled:       opts.DebugEnabled,
	}

	if store.debugEnabled {
		store.EnableDebug(true)
	}

	if store.automigrateEnabled {
		err := store.MigrateUp(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

// MigrateUp creates the cache table
func (st *storeImplementation) MigrateUp(ctx context.Context, tx ...*sql.Tx) error {
	if st.db.Schema().HasTable(st.cacheTableName) {
		if st.debugEnabled {
			st.logger.Info("MigrateUp: table already exists", "table", st.cacheTableName)
		}
		return nil
	}

	err := st.db.Schema().Create(st.cacheTableName, func(table contractsschema.Blueprint) {
		table.String(COLUMN_ID, 21)
		table.Primary(COLUMN_ID)
		table.String(COLUMN_KEY, 255)
		table.Text(COLUMN_VALUE)
		table.DateTime(COLUMN_EXPIRES_AT)
		table.DateTime(COLUMN_CREATED_AT)
		table.DateTime(COLUMN_UPDATED_AT)
		table.DateTime(constants.SoftDeleteAtColumn).Default(constants.MaxSoftDeletedAtDefault)
	})

	if err != nil {
		if st.debugEnabled {
			st.logger.Error("MigrateUp failed", "error", err)
		}
		return err
	}

	return nil
}

// MigrateDown drops the cache table
func (st *storeImplementation) MigrateDown(ctx context.Context, tx ...*sql.Tx) error {
	if !st.db.Schema().HasTable(st.cacheTableName) {
		if st.debugEnabled {
			st.logger.Info("MigrateDown: table does not exist", "table", st.cacheTableName)
		}
		return nil
	}

	err := st.db.Schema().Drop(st.cacheTableName)
	if err != nil {
		if st.debugEnabled {
			st.logger.Error("MigrateDown failed", "error", err)
		}
		return err
	}
	return nil
}

// EnableDebug enables or disables debug mode
func (st *storeImplementation) EnableDebug(debug bool) {
	st.debugEnabled = debug
	if debug {
		st.db.EnableDebug()
		st.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	} else {
		st.db.DisableDebug()
		st.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
}

// GetCacheTableName returns the cache table name
func (st *storeImplementation) GetCacheTableName() string {
	return st.cacheTableName
}

// SetCacheTableName sets the cache table name
func (st *storeImplementation) SetCacheTableName(cacheTableName string) {
	st.cacheTableName = cacheTableName
}

// ExpireCacheGoroutine soft deletes expired cache using the provided context
// for cancellation.
func (st *storeImplementation) ExpireCacheGoroutine(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := st.expireCacheOnce(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(60 * time.Second):
			if err := st.expireCacheOnce(); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			}
		}
	}
}

func (st *storeImplementation) expireCacheOnce() error {
	if st.debugEnabled {
		st.logger.Debug("Cleaning expired cache...")
	}

	_, err := st.db.Query().
		Model(&Cache{}).
		Table(st.cacheTableName).
		Where(COLUMN_EXPIRES_AT+" < ?", carbon.Now(carbon.UTC).StdTime()).
		Delete()

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		if st.debugEnabled {
			st.logger.Debug("expireCacheOnce error", "error", err)
		}
		// Don't return error for cleanup failures
		return nil
	}

	return nil
}

// FindByKey finds a cache by key
func (st *storeImplementation) FindByKey(key string) (*Cache, error) {
	if key == "" {
		return nil, nil
	}

	var cache Cache
	err := st.db.Query().
		Model(&Cache{}).
		Table(st.cacheTableName).
		Where(COLUMN_KEY+" = ?", key).
		Where(COLUMN_EXPIRES_AT+" > ?", carbon.Now(carbon.UTC).StdTime()).
		First(&cache)

	if err != nil {
		if err.Error() == "no rows found" {
			return nil, nil
		}
		if st.debugEnabled {
			st.logger.Debug("FindByKey error", "error", err, "key", key)
		}
		return nil, err
	}

	return &cache, nil
}

// Get gets a key from cache
func (st *storeImplementation) Get(key string, valueDefault string) (string, error) {
	cache, errFind := st.FindByKey(key)

	if errFind != nil {
		return valueDefault, errFind
	}

	if cache != nil {
		return cache.ValueField, nil
	}

	return valueDefault, nil
}

// GetJSON gets a JSON key from cache
func (st *storeImplementation) GetJSON(key string, valueDefault interface{}) (interface{}, error) {
	cache, errFind := st.FindByKey(key)

	if errFind != nil {
		return valueDefault, errFind
	}

	if cache != nil {
		jsonValue := cache.ValueField
		var e interface{}
		jsonError := json.Unmarshal([]byte(jsonValue), &e)
		if jsonError != nil {
			return valueDefault, jsonError
		}

		return e, nil
	}

	return valueDefault, nil
}

// Remove removes a key from cache
func (st *storeImplementation) Remove(key string) error {
	if key == "" {
		return nil
	}

	_, err := st.db.Query().
		Model(&Cache{}).
		Table(st.cacheTableName).
		Where(COLUMN_KEY+" = ?", key).
		Delete()

	if err != nil {
		if st.debugEnabled {
			st.logger.Debug("Remove error", "error", err, "key", key)
		}
		// Return nil for not found cases
		return nil
	}

	return nil
}

// Set sets new key value pair
func (st *storeImplementation) Set(key string, value string, seconds int64) error {
	if key == "" {
		return errors.New("key is required")
	}

	cache, errFind := st.FindByKey(key)

	if errFind != nil {
		return errFind
	}

	expiresAt := time.Now().Add(time.Second * time.Duration(seconds))

	if cache == nil {
		// Create new cache entry
		row := map[string]any{
			COLUMN_ID:         neatuid.GenerateShortID(),
			COLUMN_KEY:        key,
			COLUMN_VALUE:      value,
			COLUMN_EXPIRES_AT: expiresAt,
			COLUMN_CREATED_AT: carbon.Now(carbon.UTC).StdTime(),
			COLUMN_UPDATED_AT: carbon.Now(carbon.UTC).StdTime(),
			COLUMN_DELETED_AT: soft_delete.MaxSoftDeletedAt,
		}

		err := st.db.Query().Table(st.cacheTableName).Create(row)
		if err != nil {
			if st.debugEnabled {
				st.logger.Debug("Set create error", "error", err)
			}
			return err
		}
	} else {
		// Update existing cache entry
		row := map[string]any{
			COLUMN_VALUE:      value,
			COLUMN_EXPIRES_AT: expiresAt,
			COLUMN_UPDATED_AT: carbon.Now(carbon.UTC).StdTime(),
		}

		_, err := st.db.Query().
			Model(&Cache{}).
			Table(st.cacheTableName).
			Where(COLUMN_ID+" = ?", cache.ID).
			Update(row)

		if err != nil {
			if st.debugEnabled {
				st.logger.Debug("Set update error", "error", err)
			}
			return err
		}
	}

	return nil
}

// SetJSON sets a JSON value in cache
func (st *storeImplementation) SetJSON(key string, value any, seconds int64) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return st.Set(key, string(jsonValue), seconds)
}
