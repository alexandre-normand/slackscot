package store

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"path/filepath"
)

type Store struct {
	Name     string
	database *leveldb.DB
}

func NewStore(name string, storagePath string) (store *Store, err error) {
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

	return &Store{name, db}, nil
}

func (store *Store) Close() {
	store.database.Close()
}

func (store *Store) Get(key string) (value string, err error) {
	data, err := store.database.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (store *Store) Put(key string, value string) (err error) {
	return store.database.Put([]byte(key), []byte(value), nil)
}

func (store *Store) Scan() (entries map[string]string, err error) {
	entries = map[string]string{}
	iter := store.database.NewIterator(nil, nil)
	for iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())
		entries[key] = value
	}

	iter.Release()
	err = iter.Error()

	return entries, err
}
