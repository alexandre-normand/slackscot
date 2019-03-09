package datastoredb

import (
	"cloud.google.com/go/datastore"
	"context"
	"google.golang.org/api/option"
	"io"
)

// gcdatastore wraps an actual google cloud datastore Client for real/production datastore interaction
type gcdatastore struct {
	*datastore.Client
	gcloudProjectID  string
	gcloudClientOpts []option.ClientOption
}

// connecter is implemented by any value that has a connect method
type connecter interface {
	connect() (err error)
}

// connect creates a new client instance from the initial gcloud project id and client options
// If the client options can be updated during the course of a process (such as option.WithCredentialsFile),
// connect should be able to reflect changes in those when it lazily reconnects on error
func (ds *gcdatastore) connect() (err error) {
	ctx := context.Background()

	ds.Client, err = datastore.NewClient(ctx, ds.gcloudProjectID, ds.gcloudClientOpts...)
	if err != nil {
		return err
	}

	return nil
}

// datastorer is implemented by any value that implements all of its methods. It is meant
// to allow easier testing decoupled from an actual datastore to interact with and
// the methods defined are method implemented by the datastore.Client that this package
// uses
type datastorer interface {
	connecter
	io.Closer
	Delete(c context.Context, k *datastore.Key) (err error)
	Get(c context.Context, k *datastore.Key, dest interface{}) (err error)
	GetAll(c context.Context, query interface{}, dest interface{}) (keys []*datastore.Key, err error)
	Put(c context.Context, k *datastore.Key, v interface{}) (key *datastore.Key, err error)
}

// Delete deletes the entity for the given key. See https://godoc.org/cloud.google.com/go/datastore#Client.Delete
func (ds *gcdatastore) Delete(c context.Context, k *datastore.Key) (err error) {
	return ds.Delete(c, k)
}

// Get loads the entity stored for key into dst. See https://godoc.org/cloud.google.com/go/datastore#Client.Get
func (ds *gcdatastore) Get(c context.Context, k *datastore.Key, dest interface{}) (err error) {
	return ds.Get(c, k, dest)
}

// GetAll runs the provided query in the given context and returns all keys that match that query.
// See https://godoc.org/cloud.google.com/go/datastore#Client.GetAll
func (ds *gcdatastore) GetAll(c context.Context, query interface{}, dest interface{}) (keys []*datastore.Key, err error) {
	return ds.GetAll(c, query, dest)
}

// Put saves the entity src into the datastore with the given key. See https://godoc.org/cloud.google.com/go/datastore#Client.Put
func (ds *gcdatastore) Put(c context.Context, k *datastore.Key, v interface{}) (key *datastore.Key, err error) {
	return ds.Put(c, k, v)
}
