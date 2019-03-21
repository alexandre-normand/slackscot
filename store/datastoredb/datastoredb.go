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
	datastorer
	kind string
}

// EntryValue represents an entity/entry value mapped to a datastore key
type EntryValue struct {
	Value string `datastore:",noindex"`
}

const (
	// Try operations that could fail at most twice. The first time is assummed to potentially fail because
	// of authentication errors when credentials have expired. The second time, a failure is probably
	// something to report back
	maxAttemptCount = 2
)

// New returns a new instance of DatastoreDB for the given name (which maps to the datastore entity "Kind" and can
// be thought of as the namespace). This function also requires a gcloudProjectID as well as at least one option to provide gcloud client credentials.
// Note that in order to support a deployment where credentials can get updated, the gcloudClientOpts should use
// something like option.WithCredentialsFile with the credentials file being updated on disk so that when reconnecting
// on a failure, the updated credentials are visible through the same gcloud client options
func New(name string, gcloudProjectID string, gcloudClientOpts ...option.ClientOption) (dsdb *DatastoreDB, err error) {
	ds := new(gcdatastore)
	ds.gcloudProjectID = gcloudProjectID
	ds.gcloudClientOpts = gcloudClientOpts

	return newWithDatastorer(name, ds)
}

// newWithDatastorer returns a new instance of DatastoreDB using the provided datastorer
func newWithDatastorer(name string, datastorer datastorer) (dsdb *DatastoreDB, err error) {
	dsdb = new(DatastoreDB)
	dsdb.kind = name
	dsdb.datastorer = datastorer

	err = dsdb.connect()
	if err != nil {
		return nil, err
	}

	err = dsdb.testDB()
	if err = dsdb.testDB(); err != nil {
		dsdb.Close()
		return nil, err
	}

	return dsdb, nil
}

// testDB makes a lightweight call to the datastore to validate connectivity and credentials
func (dsdb *DatastoreDB) testDB() (err error) {
	err = dsdb.Get(context.Background(), datastore.NameKey(dsdb.kind, "testConnectivity", nil), &EntryValue{})

	if err != nil && err != datastore.ErrNoSuchEntity {
		return err
	}

	return nil
}

// GetString returns the value associated to a given key. If the value is not
// found or an error occurred, the zero-value string is returned along with
// the error
func (dsdb *DatastoreDB) GetString(key string) (value string, err error) {
	return dsdb.GetSiloString("", key)
}

// GetSiloString returns the value associated to a given key within the silo provided. If the value is not
// found or an error occurred, the zero-value string is returned along with the error
func (dsdb *DatastoreDB) GetSiloString(silo string, key string) (value string, err error) {
	ctx := context.Background()

	var e EntryValue
	k := newKeyWithNamespace(silo, dsdb.kind, key)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	var attempt int
	for attempt, err = 1, dsdb.Get(ctx, k, &e); attempt < maxAttemptCount && err != nil && shouldRetry(err); attempt, err = attempt+1, dsdb.Get(ctx, k, &e) {
		dsdb.connect()
	}

	if err != nil {
		return "", err
	}

	return e.Value, nil
}

// PutString stores the key/value to the database
func (dsdb *DatastoreDB) PutString(key string, value string) (err error) {
	return dsdb.PutSiloString("", key, value)
}

// PutSiloString stores the key/value to the database in the given silo
func (dsdb *DatastoreDB) PutSiloString(silo string, key string, value string) (err error) {
	ctx := context.Background()
	k := newKeyWithNamespace(silo, dsdb.kind, key)

	// Execute first attempt
	_, err = dsdb.Put(ctx, k, &EntryValue{Value: value})

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt := 1; attempt < maxAttemptCount && err != nil && shouldRetry(err); attempt = attempt + 1 {
		dsdb.connect()

		_, err = dsdb.Put(ctx, k, &EntryValue{Value: value})
	}

	return err
}

// DeleteString deletes the entry for the given key. If the entry is not found
// an error is returned
func (dsdb *DatastoreDB) DeleteString(key string) (err error) {
	return dsdb.DeleteSiloString("", key)
}

// DeleteSiloString deletes the entry for the given key in the given silo. If the entry is not found
// an error is returned
func (dsdb *DatastoreDB) DeleteSiloString(silo string, key string) (err error) {
	ctx := context.Background()
	k := newKeyWithNamespace(silo, dsdb.kind, key)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	var attempt int
	for attempt, err = 1, dsdb.Delete(ctx, k); attempt < maxAttemptCount && err != nil && shouldRetry(err); attempt, err = attempt+1, dsdb.Delete(ctx, k) {
		dsdb.connect()
	}

	return err
}

// Scan returns all key/values from the database
func (dsdb *DatastoreDB) Scan() (entries map[string]string, err error) {
	return dsdb.ScanSilo("")
}

// ScanSilo returns all key/values from the database in the given silo
func (dsdb *DatastoreDB) ScanSilo(silo string) (entries map[string]string, err error) {
	entries = make(map[string]string)

	keys, vals, err := dsdb.scan(datastore.NewQuery(dsdb.kind).Namespace(silo))

	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		entries[key.Name] = vals[i].Value
	}

	return entries, nil
}

// GlobalScan returns all key/values for all silos keyed by silo name
func (dsdb *DatastoreDB) GlobalScan() (entries map[string]map[string]string, err error) {
	entries = make(map[string]map[string]string)

	keys, vals, err := dsdb.scan(datastore.NewQuery(dsdb.kind))
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		if _, ok := entries[key.Namespace]; !ok {
			entries[key.Namespace] = make(map[string]string)
		}

		entries[key.Namespace][key.Name] = vals[i].Value
	}

	return entries, nil
}

// scan runs an internal datastore query and returns the raw keys and values for post-processing
func (dsdb *DatastoreDB) scan(query *datastore.Query) (keys []*datastore.Key, vals []*EntryValue, err error) {
	ctx := context.Background()

	vals = make([]*EntryValue, 0)

	// Run first attempt before looping
	keys, err = dsdb.GetAll(ctx, query, &vals)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt := 1; attempt < maxAttemptCount && err != nil && shouldRetry(err); attempt = attempt + 1 {
		dsdb.connect()

		keys, err = dsdb.GetAll(ctx, query, &vals)
	}

	return keys, vals, err
}

// shouldRetry returns true if the given error should be retried or false if not.
// In order to determine this, one approach would be to only retry on a
// statusError (https://github.com/grpc/grpc-go/blob/master/status/status.go#L43)
// with code Unauthenticated (https://godoc.org/google.golang.org/grpc/codes) but that's made
// trickier by the statusError not being promoted outside the package (checking for the Error string
// would be reasonable but a bit dirty).
// Alternatively, and what's done here is to be a little conservative and retry on everything except
// ErrNoSuchEntity, ErrInvalidEntityType and ErrInvalidKey which are not things retries would help
// with. This means we could still retry when it's pointless to do so at the expense of added latency.
func shouldRetry(err error) bool {
	return err != datastore.ErrNoSuchEntity && err != datastore.ErrInvalidEntityType && err != datastore.ErrInvalidKey
}

// newKeyWithNamespace returns a new datastore key for the given kind and key name within the
// specified namespace
func newKeyWithNamespace(namespace string, kind string, key string) (k *datastore.Key) {
	k = datastore.NameKey(kind, key, nil)
	k.Namespace = namespace

	return k
}
