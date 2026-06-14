package cachestore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func initTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:?parseTime=true")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestStoreCreate(t *testing.T) {
	db := initTestDB(t)

	store, err := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_create",
		AutomigrateEnabled: true,
	})

	if err != nil {
		t.Fatalf("Store could not be created: %v", err)
	}

	if store == nil {
		t.Fatalf("Store could not be created")
	}

	err = store.Set("post", "1234567890", 5)

	if err != nil {
		t.Fatalf("Cache could not be created: %v", err)
	}
}

func TestStoreAutomigrate(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_automigrate",
		AutomigrateEnabled: false,
	})

	err := store.MigrateUp(context.Background())

	if err != nil {
		t.Fatal("MigrateUp failed: " + err.Error())
	}

	err = store.Set("post", "1234567890", 5)

	if err != nil {
		t.Fatalf("Cache could not be created: %v", err)
	}
}

func TestStoreCacheDelete(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_automigrate",
		AutomigrateEnabled: true,
	})

	err := store.Remove("post")

	if err != nil {
		t.Fatalf("Entity could not be created: %v", err)
	}

	val, err := store.FindByKey("post")
	if err != nil {
		t.Fatalf("Getting JSON failed: %v", err)
	}
	if val != nil {
		t.Fatalf("Cache should no longer be present")
	}
}

func TestStoreEnableDebug(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_debug",
		AutomigrateEnabled: false,
	})
	store.EnableDebug(true)

	err := store.MigrateUp(context.Background())

	if err != nil {
		t.Fatal("MigrateUp failed: " + err.Error())
	}
}

func TestSetKey(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_set_key",
		AutomigrateEnabled: true,
	})

	err := store.Set("hello", "world", 1)

	if err != nil {
		t.Fatalf("Setting key failed: %v", err)
	}

	value, err := store.Get("hello", "")
	if err != nil {
		t.Fatalf("Getting JSON failed: %v", err)
	}

	if value != "world" {
		t.Fatalf("Incorrect value: %v", value)
	}
}

func TestUpdateKey(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_update_key",
		AutomigrateEnabled: true,
		DebugEnabled:       true,
	})

	// Use a longer TTL so the entry doesn't expire during the test
	err := store.Set("hello", "world", 60)
	if err != nil {
		t.Fatalf("Setting key failed: %v", err)
	}

	cache1, err := store.FindByKey("hello")
	if err != nil {
		t.Fatalf("Find by key failed: %v", err)
	}
	if cache1 == nil {
		t.Fatalf("Initial cache not found")
	}

	time.Sleep(2 * time.Second)

	// Update the existing entry (won't expire since first TTL was 60 seconds)
	err2 := store.Set("hello", "world", 60)
	if err2 != nil {
		t.Fatalf("Update setting key failed: %v", err2)
	}

	cache2, err := store.FindByKey("hello")
	if err != nil {
		t.Fatalf("Find by key failed: %v", err)
	}

	if cache2 == nil {
		t.Fatalf("Cache not found after second Set")
	}

	if cache2.ValueField != "world" {
		t.Fatalf("Value not correct: %s", cache2.ValueField)
	}

	if cache2.KeyField != "hello" {
		t.Fatalf("Key not correct: %s", cache2.KeyField)
	}

	// Access time.Time fields directly
	if cache2.UpdatedAtField.Equal(cache1.CreatedAtField) {
		t.Fatalf("Updated at should be different from created at date: %s", cache2.UpdatedAtField.Format(time.UnixDate))
	}

	if cache2.UpdatedAtField.Sub(cache1.CreatedAtField).Seconds() < 1 {
		t.Fatalf("Updated at should more than 1 second after created at date: %s - %s",
			cache2.UpdatedAtField.Format(time.UnixDate),
			cache1.CreatedAtField.Format(time.UnixDate))
	}
}

func TestSetGetJSON(t *testing.T) {
	db := initTestDB(t)

	store, _ := NewStore(NewStoreOptions{
		DB:                 db,
		CacheTableName:     "cache_automigrate",
		AutomigrateEnabled: true,
	})

	err := store.SetJSON("hello", map[string]string{"first_name": "Jo"}, 1)

	if err != nil {
		t.Fatalf("Setting key failed: %v", err)
	}

	value, err := store.GetJSON("hello", "")

	if err != nil {
		t.Fatalf("Getting JSON failed: %v", err)
	}

	result := value.(map[string]interface{})
	if result["first_name"] != "Jo" {
		t.Fatalf("Incorrect value: %s", value)
	}
}
