package datastoredb

import (
	"cloud.google.com/go/datastore"
	"context"
	"google.golang.org/api/option"
)

// DatastoreDB implements the slackscot StringStorer interface. It maps
// the given name (usually a plugin name) to the datastore entity Kind
// to isolate data between different plugins
type DatastoreDB struct {
	*datastore.Client
	kind string
}

// EntryValue represents an entity/entry value mapped to a datastore key
type EntryValue struct {
	Value string `datastore:",noindex"`
}

// NewDatastoreDB returns a new instance of DatastoreDB for the given name (which maps to the datastore entity "Kind" and can
// be thought of as the namespace). This function also requires a gcloudProjectID as well as at least one option to provide gcloud client credentials
func NewDatastoreDB(name string, gcloudProjectID string, gcloudClientOpts ...option.ClientOption) (dsdb *DatastoreDB, err error) {
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, gcloudProjectID, gcloudClientOpts...)
	if err != nil {
		return nil, err
	}

	dsdb = new(DatastoreDB)
	dsdb.Client = client
	dsdb.kind = name

	err = dsdb.testDB()
	if err = dsdb.testDB(); err != nil {
		dsdb.Close()
		return nil, err
	}

	return dsdb, nil
}

// testDB makes a lightweight call to the datastore to validate connectivity and credentials
func (dsdb *DatastoreDB) testDB() (err error) {
	_, err = dsdb.GetString("testConnectivity")

	if err != nil && err != datastore.ErrNoSuchEntity {
		return err
	}

	return nil
}

// GetString returns the value associated to a given key. If the value is not
// found or an error occured, the zero-value string is returned along with
// the error
func (dsdb *DatastoreDB) GetString(key string) (value string, err error) {
	ctx := context.Background()

	var e EntryValue
	k := datastore.NameKey(dsdb.kind, key, nil)
	if err := dsdb.Get(ctx, k, &e); err != nil {
		return "", err
	}

	return e.Value, nil
}

// PutString stores the key/value to the database
func (dsdb *DatastoreDB) PutString(key string, value string) (err error) {
	ctx := context.Background()
	k := datastore.NameKey(dsdb.kind, key, nil)

	_, err = dsdb.Put(ctx, k, &EntryValue{Value: value})
	return err
}

// DeleteString deletes the entry for the given key. If the entry is not found
// an error is returned
func (dsdb *DatastoreDB) DeleteString(key string) (err error) {
	ctx := context.Background()
	k := datastore.NameKey(dsdb.kind, key, nil)

	return dsdb.Delete(ctx, k)
}

// Scan returns all key/values from the database
func (dsdb *DatastoreDB) Scan() (entries map[string]string, err error) {
	entries = make(map[string]string)

	ctx := context.Background()
	var vals []*EntryValue

	keys, err := dsdb.GetAll(ctx, datastore.NewQuery(dsdb.kind), &vals)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		entries[key.Name] = vals[i].Value
	}

	return entries, nil
}

// Close closes the underlying datastore client
func (dsdb *DatastoreDB) Close() (err error) {
	return dsdb.Client.Close()
}
