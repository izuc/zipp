package database

import (
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/kvstore/mapdb"
)

type memDB struct {
	kvstore.KVStore
}

// NewMemDB returns a new in-memory (not persisted) DB object.
func NewMemDB() (DB, error) {
	return &memDB{KVStore: mapdb.NewMapDB()}, nil
}

func (db *memDB) NewStore() kvstore.KVStore {
	return db.KVStore
}

func (db *memDB) Close() error {
	db.KVStore = nil
	return nil
}

func (db *memDB) RequiresGC() bool {
	return false
}

func (db *memDB) GC() error {
	return nil
}
