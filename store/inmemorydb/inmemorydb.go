package inmemorydb

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/store"
)

// InMemoryDB implements the slackscot StringStorer interface and keeps
// a copy of everything in memory while writing through puts and deletes
// to the wrapped (persistent) StringStorer
type InMemoryDB struct {
	persistentStorer store.StringStorer
	data             map[string]string
}

// New returns a new instance of InMemoryDB wrapping the persistent StringStorer.
// Note that instantiation might have some latency induced by the initial scan to load
// the current database content from the persistentStorer in memory
func New(storer store.StringStorer) (imdb *InMemoryDB, err error) {
	imdb = new(InMemoryDB)
	imdb.persistentStorer = storer

	imdb.data, err = imdb.persistentStorer.Scan()
	if err != nil {
		return nil, err
	}

	return imdb, nil
}

// GetString returns the value associated to a given key. If the value is not
// found or an error occurred, the zero-value string is returned along with
// the error
func (imdb *InMemoryDB) GetString(key string) (value string, err error) {
	v, ok := imdb.data[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}

	return v, nil
}

// PutString stores the key/value to the database. The key/value is persisted to
// persistent storage and also kept in memory
func (imdb *InMemoryDB) PutString(key string, value string) (err error) {
	err = imdb.persistentStorer.PutString(key, value)

	if err != nil {
		return err
	}

	imdb.data[key] = value
	return nil
}

// DeleteString deletes the entry for the given key. This is propagated to the
// persistent storage first and then deleted from memory
func (imdb *InMemoryDB) DeleteString(key string) (err error) {
	err = imdb.persistentStorer.DeleteString(key)
	if err != nil {
		return err
	}

	delete(imdb.data, key)
	return nil
}

// Scan returns all key/values from the database. This one returns a copy of the in-memory
// copy without querying the persistent storer.
func (imdb *InMemoryDB) Scan() (entries map[string]string, err error) {
	entries = make(map[string]string)

	for k, v := range imdb.data {
		entries[k] = v
	}

	return entries, nil
}

// Close closes the underlying storer
func (imdb *InMemoryDB) Close() (err error) {
	return imdb.persistentStorer.Close()
}
