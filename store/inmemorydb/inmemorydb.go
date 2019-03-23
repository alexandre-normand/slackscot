package inmemorydb

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/store"
)

// InMemoryDB implements the slackscot GlobalSiloStringStorer interface and keeps
// a copy of everything in memory while writing through puts and deletes
// to the wrapped (persistent) GlobalSiloStringStorer
type InMemoryDB struct {
	persistentStorer store.GlobalSiloStringStorer
	data             map[string]map[string]string
}

// New returns a new instance of InMemoryDB wrapping the persistent GlobalSiloStringStorer.
// Note that instantiation might have some latency induced by the initial scan to load
// the current database content from the persistentStorer in memory
func New(storer store.GlobalSiloStringStorer) (imdb *InMemoryDB, err error) {
	imdb = new(InMemoryDB)
	imdb.persistentStorer = storer

	imdb.data, err = imdb.persistentStorer.GlobalScan()
	if err != nil {
		return nil, err
	}

	return imdb, nil
}

// GetString returns the value associated to a given key. If the value is not
// found or an error occurred, the zero-value string is returned along with
// the error
func (imdb *InMemoryDB) GetString(key string) (value string, err error) {
	return imdb.GetSiloString("", key)
}

// GetSiloString returns the value associated to a given key in the given silo.
// If the value is not found or an error occurred, the zero-value string is returned along with
// the error
func (imdb *InMemoryDB) GetSiloString(silo string, key string) (value string, err error) {
	s, ok := imdb.data[silo]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}

	v, ok := s[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}

	return v, nil
}

// PutString stores the key/value to the database. The key/value is persisted to
// persistent storage and also kept in memory
func (imdb *InMemoryDB) PutString(key string, value string) (err error) {
	return imdb.PutSiloString("", key, value)
}

// PutSiloString stores the key/value to a silo the database. The key/value is persisted to
// persistent storage and also kept in memory
func (imdb *InMemoryDB) PutSiloString(silo string, key string, value string) (err error) {
	err = imdb.persistentStorer.PutSiloString(silo, key, value)

	if err != nil {
		return err
	}

	if _, ok := imdb.data[silo]; !ok {
		imdb.data[silo] = make(map[string]string)
	}

	imdb.data[silo][key] = value
	return nil
}

// DeleteString deletes the entry for the given key. This is propagated to the
// persistent storage first and then deleted from memory
func (imdb *InMemoryDB) DeleteString(key string) (err error) {
	return imdb.DeleteSiloString("", key)
}

// DeleteSiloString deletes the silo entry for the given key. This is propagated to the
// persistent storage first and then deleted from memory
func (imdb *InMemoryDB) DeleteSiloString(silo string, key string) (err error) {
	err = imdb.persistentStorer.DeleteSiloString(silo, key)
	if err != nil {
		return err
	}

	if s, ok := imdb.data[silo]; ok {
		delete(s, key)
	}

	return nil
}

// Scan returns all key/values from the database. This one returns a copy of the in-memory
// copy without querying the persistent storer.
func (imdb *InMemoryDB) Scan() (entries map[string]string, err error) {
	return imdb.ScanSilo("")
}

// ScanSilo returns all key/values for a silo from the database. This one returns a copy of the in-memory
// copy without querying the persistent storer.
func (imdb *InMemoryDB) ScanSilo(silo string) (entries map[string]string, err error) {
	entries = make(map[string]string)

	for k, v := range imdb.data[silo] {
		entries[k] = v
	}

	return entries, nil
}

// GlobalScan returns all key/values from the database. This one returns a copy of the in-memory
// copy without querying the persistent storer.
func (imdb *InMemoryDB) GlobalScan() (entries map[string]map[string]string, err error) {
	entries = make(map[string]map[string]string)

	for s, sc := range imdb.data {
		for k, v := range sc {
			if _, ok := entries[s]; !ok {
				entries[s] = make(map[string]string)
			}

			entries[s][k] = v
		}

	}

	return entries, nil
}

// Close closes the underlying storer
func (imdb *InMemoryDB) Close() (err error) {
	return imdb.persistentStorer.Close()
}
