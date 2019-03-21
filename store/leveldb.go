package store

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
	"strings"
)

// LevelDB holds a datastore name and its leveldb instance
type LevelDB struct {
	Name     string
	database *leveldb.DB
}

const (
	siloKeyDelimiter = "\u00DA" // \xDA is not a valid UTF8 character so it serves fairly well as a delimiter for strings.
)

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

// GetSiloString retrieves a value associated to the key in the given silo
func (ldb *LevelDB) GetSiloString(silo string, key string) (value string, err error) {
	val, err := ldb.database.Get([]byte(EncodeKey(silo, key)), nil)
	if err != nil {
		return "", err
	}

	return string(val), nil
}

// GetString retrieves a value associated to the key
func (ldb *LevelDB) GetString(key string) (value string, err error) {
	return ldb.GetSiloString("", key)
}

// Get retrieves a value associated to the key
func (ldb *LevelDB) Get(key []byte) (value []byte, err error) {
	val, err := ldb.GetSiloString("", string(key))
	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}

// PutSiloString adds or updates a value associated to the key in the given silo
func (ldb *LevelDB) PutSiloString(silo string, key string, value string) (err error) {
	return ldb.database.Put([]byte(EncodeKey(silo, key)), []byte(value), nil)
}

// PutString adds or updates a value associated to the key
func (ldb *LevelDB) PutString(key string, value string) (err error) {
	return ldb.PutSiloString("", key, value)
}

// Put adds or updates a value associated to the key
func (ldb *LevelDB) Put(key []byte, value []byte) (err error) {
	return ldb.PutSiloString("", string(key), string(value))
}

// DeleteSiloString deletes an entry for a given key string in the given silo
func (ldb *LevelDB) DeleteSiloString(silo string, key string) (err error) {
	return ldb.database.Delete([]byte(EncodeKey(silo, key)), nil)
}

// DeleteString deletes an entry for a given key string
func (ldb *LevelDB) DeleteString(key string) (err error) {
	return ldb.DeleteSiloString("", key)
}

// Delete deletes an entry for a given key
func (ldb *LevelDB) Delete(key []byte) (err error) {
	return ldb.DeleteSiloString("", string(key))
}

// Scan returns the complete set of key/values from the database
func (ldb *LevelDB) Scan() (entries map[string]string, err error) {
	return ldb.ScanSilo("")
}

// EncodeKey encodes a key with the silo name and the \xda character (not a valid utf8 character)
func EncodeKey(silo string, key string) (encKey string) {
	return SiloPrefix(silo) + key
}

// SiloPrefix returns the prefix for a key in the given silo
func SiloPrefix(silo string) (prefix string) {
	return silo + siloKeyDelimiter
}

// DecodeKey returns a logical key and silo given its raw key value
func DecodeKey(rawKey string) (silo string, key string, err error) {
	parts := strings.Split(rawKey, siloKeyDelimiter)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid number of parts in key [%s], 2 expected but got [%d]", rawKey, len(parts))
	}

	return parts[0], parts[1], nil
}

// ScanSilo returns the complete set of key/values from the database in the given silo
func (ldb *LevelDB) ScanSilo(silo string) (entries map[string]string, err error) {
	entries = map[string]string{}
	iter := ldb.database.NewIterator(util.BytesPrefix([]byte(SiloPrefix(silo))), nil)
	for iter.Next() {
		_, key, err := DecodeKey(string(iter.Key()))
		if err != nil {
			return nil, err
		}

		value := string(iter.Value())
		entries[key] = value
	}

	iter.Release()
	err = iter.Error()

	return entries, err
}

// GlobalScan returns the complete set of key/values from the database for all silos
func (ldb *LevelDB) GlobalScan() (entries map[string]map[string]string, err error) {
	entries = make(map[string]map[string]string)
	iter := ldb.database.NewIterator(nil, nil)
	for iter.Next() {
		silo, key, err := DecodeKey(string(iter.Key()))
		if err != nil {
			return nil, err
		}

		value := string(iter.Value())

		if _, ok := entries[silo]; !ok {
			entries[silo] = make(map[string]string)
		}

		entries[silo][key] = value
	}

	iter.Release()
	err = iter.Error()

	return entries, err
}
