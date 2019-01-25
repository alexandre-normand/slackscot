package store

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"path/filepath"
)

// LevelDB holds a datastore name and its leveldb instance
type LevelDB struct {
	Name     string
	database *leveldb.DB
}

// NewLevelDB instantiates and open a new LevelDB instance backed by a leveldb database. If the
// leveldb database doesn't exist, one is created
func NewLevelDB(name string, storagePath string) (ldb *LevelDB, err error) {
	// Expand '~' as the full home directory path if appropriate
	path, err := homedir.Expand(storagePath)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(path, name)
	db, err := leveldb.OpenFile(fullPath, nil)

	if _, ok := err.(*leveldberrors.ErrCorrupted); ok {
		return nil, errors.Wrap(err, fmt.Sprintf("leveldb corrupted. Consider deleting [%s] and restarting if you don't mind losing data", fullPath))
	} else if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to open file with path [%s]", fullPath))
	}

	return &LevelDB{name, db}, nil
}

// Close closes the LevelDB
func (ldb *LevelDB) Close() (err error) {
	return ldb.database.Close()
}

// GetString retrieves a value associated to the key
func (ldb *LevelDB) GetString(key string) (value string, err error) {
	val, err := ldb.Get([]byte(key))

	return string(val), err
}

// Get retrieves a value associated to the key
func (ldb *LevelDB) Get(key []byte) (value []byte, err error) {
	value, err = ldb.database.Get(key, nil)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// PutString adds or updates a value associated to the key
func (ldb *LevelDB) PutString(key string, value string) (err error) {
	return ldb.database.Put([]byte(key), []byte(value), nil)
}

// Put adds or updates a value associated to the key
func (ldb *LevelDB) Put(key []byte, value []byte) (err error) {
	return ldb.database.Put(key, value, nil)
}

// Scan returns the complete set of key/values from the database
func (ldb *LevelDB) Scan() (entries map[string]string, err error) {
	entries = map[string]string{}
	iter := ldb.database.NewIterator(nil, nil)
	for iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())
		entries[key] = value
	}

	iter.Release()
	err = iter.Error()

	return entries, err
}
