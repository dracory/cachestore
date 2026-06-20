package cachestore

import (
	"time"

	"github.com/dracory/neat/database/orm"
	"github.com/dracory/neat/database/soft_delete"
)

// Cache type
type Cache struct {
	orm.ShortID
	soft_delete.SoftDeletesMaxDate

	KeyField       string     `db:"cache_key"`
	ValueField     string     `db:"cache_value"`
	ExpiresAtField *time.Time `db:"expires_at"`

	CreatedAtField time.Time `db:"created_at"`
	UpdatedAtField time.Time `db:"updated_at"`
}

// Key returns the cache key
func (c *Cache) Key() string {
	return c.KeyField
}

// SetKey sets the cache key
func (c *Cache) SetKey(key string) {
	c.KeyField = key
}

// Value returns the cache value
func (c *Cache) Value() string {
	return c.ValueField
}

// SetValue sets the cache value
func (c *Cache) SetValue(value string) {
	c.ValueField = value
}

// ExpiresAt returns the expiration time
func (c *Cache) ExpiresAt() *time.Time {
	return c.ExpiresAtField
}

// SetExpiresAt sets the expiration time
func (c *Cache) SetExpiresAt(t *time.Time) {
	c.ExpiresAtField = t
}

// CreatedAt returns the creation timestamp
func (c *Cache) CreatedAt() time.Time {
	return c.CreatedAtField
}

// UpdatedAt returns the last update timestamp
func (c *Cache) UpdatedAt() time.Time {
	return c.UpdatedAtField
}

// DeletedAt returns the soft delete timestamp (nil if not deleted)
func (c *Cache) DeletedAt() *time.Time {
	if c.SoftDeletedAt.IsZero() || !c.SoftDeletedAt.After(time.Now()) {
		return nil
	}
	t := c.SoftDeletedAt
	return &t
}
