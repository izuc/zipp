package database

import (
	"runtime"

	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/kvstore/badger"
)

type badgerDB struct {
	store kvstore.KVStore // Use the exported KVStore interface
}

// NewDB returns a new persisting DB object.
func NewDB(dirname string) (DB, error) {
	rawDB, err := badger.CreateDB(dirname)
	if err != nil {
		return nil, err
	}
	store := badger.New(rawDB)
	return &badgerDB{store: store}, nil
}

func (db *badgerDB) NewStore() kvstore.KVStore {
	return db.store // Since badger.New already returns an instance of kvstore.KVStore, just return the stored instance.
}

// Close closes a DB. It's crucial to call it to ensure all the pending updates make their way to disk.
func (db *badgerDB) Close() error {
	if err := db.store.Flush(); err != nil {
		return err
	}

	return db.store.Close()
}

func (db *badgerDB) RequiresGC() bool {
	return true
}

func (db *badgerDB) GC() error {
	// trigger the go garbage collector to release the used memory
	runtime.GC()
	return nil
}
