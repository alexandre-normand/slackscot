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
	kind             string
	gcloudProjectID  string
	gcloudClientOpts []option.ClientOption
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
	dsdb = new(DatastoreDB)
	dsdb.kind = name
	dsdb.gcloudProjectID = gcloudProjectID
	dsdb.gcloudClientOpts = gcloudClientOpts

	dsdb.connectClient()

	err = dsdb.testDB()
	if err = dsdb.testDB(); err != nil {
		dsdb.Close()
		return nil, err
	}

	return dsdb, nil
}

// connectClient establishes a new datastore Client connection
// Note: must be called after gcloudProjectID and gCloudClientOpts has been set
func (dsdb *DatastoreDB) connectClient() (err error) {
	ctx := context.Background()
	dsdb.Client, err = datastore.NewClient(ctx, dsdb.gcloudProjectID, dsdb.gcloudClientOpts...)
	if err != nil {
		return err
	}

	return nil
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
// found or an error occurred, the zero-value string is returned along with
// the error
func (dsdb *DatastoreDB) GetString(key string) (value string, err error) {
	ctx := context.Background()

	var e EntryValue
	k := datastore.NameKey(dsdb.kind, key, nil)

	attempt := 0
	err = dsdb.Get(ctx, k, &e)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt < maxAttemptCount && err != nil && shouldRetry(err) {
		dsdb.connectClient()

		attempt = attempt + 1
		err = dsdb.Get(ctx, k, &e)
	}

	if err != nil {
		return "", err
	}

	return e.Value, nil
}

// PutString stores the key/value to the database
func (dsdb *DatastoreDB) PutString(key string, value string) (err error) {
	ctx := context.Background()
	k := datastore.NameKey(dsdb.kind, key, nil)

	attempt := 0
	_, err = dsdb.Put(ctx, k, &EntryValue{Value: value})

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt < maxAttemptCount && err != nil && shouldRetry(err) {
		dsdb.connectClient()

		attempt = attempt + 1
		_, err = dsdb.Put(ctx, k, &EntryValue{Value: value})
	}

	return err
}

// DeleteString deletes the entry for the given key. If the entry is not found
// an error is returned
func (dsdb *DatastoreDB) DeleteString(key string) (err error) {
	ctx := context.Background()
	k := datastore.NameKey(dsdb.kind, key, nil)

	attempt := 0
	err = dsdb.Delete(ctx, k)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt < maxAttemptCount && err != nil && shouldRetry(err) {
		dsdb.connectClient()

		attempt = attempt + 1
		err = dsdb.Delete(ctx, k)
	}

	return err
}

// Scan returns all key/values from the database
func (dsdb *DatastoreDB) Scan() (entries map[string]string, err error) {
	entries = make(map[string]string)

	ctx := context.Background()
	var vals []*EntryValue

	attempt := 0
	keys, err := dsdb.GetAll(ctx, datastore.NewQuery(dsdb.kind), &vals)

	// Retry once and try a reconnect if the error is recoverable (like unauthenticated error)
	for attempt < maxAttemptCount && err != nil && shouldRetry(err) {
		dsdb.connectClient()
		attempt = attempt + 1
		keys, err = dsdb.GetAll(ctx, datastore.NewQuery(dsdb.kind), &vals)
	}

	for i, key := range keys {
		entries[key.Name] = vals[i].Value
	}

	return entries, nil
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

// Close closes the underlying datastore client
func (dsdb *DatastoreDB) Close() (err error) {
	return dsdb.Client.Close()
}
