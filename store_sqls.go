package cachestore

import (
	"log"

	"github.com/dracory/sb"
)

// sqlCreateTable returns a SQL string for creating the cache table
func (st *storeImplementation) sqlCreateTable() string {
	sql, err := sb.NewBuilder(st.dbDriverName).
		Table(st.cacheTableName).
		Column(sb.Column{
			Name:       "id",
			Type:       sb.COLUMN_TYPE_STRING,
			Length:     40,
			PrimaryKey: true,
		}).
		Column(sb.Column{
			Name:   "cache_key",
			Type:   sb.COLUMN_TYPE_STRING,
			Length: 255,
		}).
		Column(sb.Column{
			Name: "cache_value",
			Type: sb.COLUMN_TYPE_TEXT,
		}).
		Column(sb.Column{
			Name: "expires_at",
			Type: sb.COLUMN_TYPE_DATETIME,
		}).
		Column(sb.Column{
			Name: "created_at",
			Type: sb.COLUMN_TYPE_DATETIME,
		}).
		Column(sb.Column{
			Name: "updated_at",
			Type: sb.COLUMN_TYPE_DATETIME,
		}).
		Column(sb.Column{
			Name:     "deleted_at",
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: true,
		}).
		CreateIfNotExists()

	if err != nil {
		log.Println(err)
		return ""
	}

	return sql
}

// sqlDropTable returns a SQL string for dropping the cache table
func (st *storeImplementation) sqlDropTable() string {
	sql, err := sb.NewBuilder(st.dbDriverName).
		Table(st.cacheTableName).
		Drop()

	if err != nil {
		log.Println(err)
		return ""
	}

	return sql
}
