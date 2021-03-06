package datastoredb

import (
	"cloud.google.com/go/datastore"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// mock of the datastore
type mockDatastore struct {
	mock.Mock
	returnNoErrOnRepeatedKey bool   // If set, the mock will ignore any expected error set on a repeated invocation with the same key. Note that the key tracking is shared across all functions
	lastKey                  string // Used to keep track of the last key in order to honor the returnNoErrOnRepeatedKey and *not* return an error on the second call with the same key
}

// connect mocks a datastore connect call
func (md *mockDatastore) connect() (err error) {
	args := md.Called()

	return args.Error(0)
}

// Close mocks a datastore Close
func (md *mockDatastore) Close() (err error) {
	args := md.Called()
	return args.Error(0)
}

// Delete mocks a Delete datastore call
func (md *mockDatastore) Delete(c context.Context, k *datastore.Key) (err error) {
	args := md.Called(c, k)

	if md.lastKey == k.Name && md.returnNoErrOnRepeatedKey {
		return nil
	}

	md.lastKey = k.Name

	return args.Error(0)
}

// Get mocks a Get datastore call
func (md *mockDatastore) Get(c context.Context, k *datastore.Key, dest interface{}) (err error) {
	args := md.Called(c, k, dest)

	if e, ok := dest.(*EntryValue); ok {
		e.Value = fmt.Sprintf("val:%s", k.Name)
	}

	if md.lastKey == k.Name && md.returnNoErrOnRepeatedKey {
		return nil
	}

	md.lastKey = k.Name
	return args.Error(0)
}

// GetAll mocks a GetAll datastore call. The base was inspired by the generated mock implementation via https://github.com/vektra/mockery
// to get an idea of how to support returner functions
func (md *mockDatastore) GetAll(c context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key, err error) {
	ret := md.Called(c, query, dest)

	var r0 []*datastore.Key
	if rf, ok := ret.Get(0).(func(context.Context, *datastore.Query, interface{}) []*datastore.Key); ok {
		r0 = rf(c, query, dest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*datastore.Key)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datastore.Query, interface{}) error); ok {
		r1 = rf(c, query, dest)
	} else {
		r1 = ret.Error(1)
	}

	// Since we don't have a key for GetAll, we support the same logic as the other
	// functions by using "getAll" as the key
	if md.lastKey == "getAll" && md.returnNoErrOnRepeatedKey {
		return keys, nil
	}

	md.lastKey = "getAll"
	return r0, r1
}

// Put mocks a Put datastore call
func (md *mockDatastore) Put(c context.Context, k *datastore.Key, v interface{}) (key *datastore.Key, err error) {
	args := md.Called(c, k, v)

	if k, ok := args.Get(0).(*datastore.Key); ok {
		key = k
	}

	if md.lastKey == k.Name && md.returnNoErrOnRepeatedKey {
		return key, nil
	}

	md.lastKey = k.Name
	return key, args.Error(1)
}

func TestErrorOnCreationConnect(t *testing.T) {
	mock := mockDatastore{}
	mock.On("connect").Return(fmt.Errorf("invalid credentials"))

	_, err := newWithDatastorer("test", &mock)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials", err.Error())
	}
}

const (
	testEntityName = "chickadee"
)

func TestErrorOnDBTestOnCreation(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(fmt.Errorf("invalid credentials")).Twice()
	mockDS.On("Close").Return(nil)

	_, err := newWithDatastorer(testEntityName, &mockDS)
	if assert.Error(t, err) {
		assert.Equal(t, "invalid credentials", err.Error())
	}
}

func TestExpectedErrNoEntityOnCreationDBTest(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	assert.NotNil(t, dsdb)
}

func TestSuccessfulGetString(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.GetString("renée")
		assert.NoError(t, err)
		assert.Equal(t, "val:renée", v)
	}
}

func TestGetSiloString(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Get", mock.Anything, newKeyWithNamespace("channelSilo", testEntityName, "renée"), mock.Anything).Return(nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.GetSiloString("channelSilo", "renée")
		assert.NoError(t, err)
		assert.Equal(t, "val:renée", v)
	}
}

func TestReconnectOnGetFailure(t *testing.T) {
	// Very importantly, we set up our mock to *not* return an error on repeated calls for the same key
	mockDS := mockDatastore{returnNoErrOnRepeatedKey: true}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.GetString("renée")
		assert.NoError(t, err)
		assert.Equal(t, "val:renée", v)
	}
}

func TestFailureToGetAfterReconnectOnFailure(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		_, err := dsdb.GetString("renée")
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}

func TestSuccessfulScan(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	vals := make([]*EntryValue, 0)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName), &vals).Return(newScanReturner(map[string]map[string]string{"": {"renée": "bird"}}), nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.Scan()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"renée": "bird"}, v)
	}
}

func TestSiloScan(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	vals := make([]*EntryValue, 0)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName).Namespace("myLittleChannel"), &vals).Return(newScanReturner(map[string]map[string]string{"": {"renée": "bird"}}), nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.ScanSilo("myLittleChannel")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"renée": "bird"}, v)
	}
}

func TestGlobalScan(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery("__namespace__").KeysOnly(), nil).Return(newScanKeysReturner([]string{"ns1", "ns2"}), nil)
	ns1Vals := make([]*EntryValue, 0)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName).Namespace("ns1"), &ns1Vals).Return(newScanReturner(map[string]map[string]string{"ns1": {"renée": "bird"}}), nil)
	ns2Vals := make([]*EntryValue, 0)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName).Namespace("ns2"), &ns2Vals).Return(newScanReturner(map[string]map[string]string{"ns2": {"renée": "fish"}}), nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.GlobalScan()
		assert.NoError(t, err)
		assert.Equal(t, map[string]map[string]string{"ns1": {"renée": "bird"}, "ns2": {"renée": "fish"}}, v)
	}
}

// newScanReturner builds a mock function that will return scan data.
// This is tricky to test since we're mocking a call that takes a pointer to an array to be written to by the call 😅. To do this, we
// have set up GetAll to allow a "returner" function to be passed in for each argument. Each one of those functions must have
// the same input signature and only the return value expected at that index. I know, I know 😱. So here, we use a return function
// and set that one value for the key that we're returning in the output. This should be much easier but the datastore API is
// not the most elegant in that regards so that's just something to deal with
func newScanReturner(siloedEntries map[string]map[string]string) func(c context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key) {
	return func(c context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key) {
		if vals, ok := dest.(*[]*EntryValue); ok {
			if vals != nil {
				keys = make([]*datastore.Key, 0)
				(*vals) = make([]*EntryValue, 0)

				i := 0
				for s, entries := range siloedEntries {
					for k, v := range entries {
						keys = append(keys, newKeyWithNamespace(s, testEntityName, k))
						(*vals) = append(*vals, &EntryValue{Value: v})
						i = i + 1
					}
				}
				return keys
			}
		}

		return nil
	}
}

// newScanKeysReturner builds a mock function that will return a key scan result (which is only keys)
func newScanKeysReturner(keyNames []string) func(c context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key) {
	return func(c context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key) {
		keys = make([]*datastore.Key, 0)

		for _, kn := range keyNames {
			keys = append(keys, newKeyWithNamespace(kn, "", kn))
		}

		return keys
	}
}

func TestReconnectOnGetAllFailure(t *testing.T) {
	// Very importantly, we set up our mock to *not* return an error on repeated calls to getall
	mockDS := mockDatastore{returnNoErrOnRepeatedKey: true}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName), mock.Anything).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		v, err := dsdb.Scan()
		assert.NoError(t, err)
		assert.Empty(t, v)
	}
}

func TestFailureToGetAllAfterReconnectOnFailure(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity).Twice()
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName), mock.Anything).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated")).Twice()

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		_, err := dsdb.Scan()
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}

func TestRepeatedFailuresOnGlobalScan(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity).Twice()
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery("__namespace__").KeysOnly(), nil).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated")).Twice()

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		_, err := dsdb.GlobalScan()
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}

func TestRepeatedFailureScanningSiloOnGlobalScan(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity).Twice()
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery("__namespace__").KeysOnly(), nil).Return(newScanKeysReturner([]string{"ns1"}), nil)
	mockDS.On("GetAll", mock.Anything, datastore.NewQuery(testEntityName).Namespace("ns1"), mock.Anything).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated")).Twice()

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		_, err := dsdb.GlobalScan()
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}

func TestPutSuccessful(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Put", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(datastore.NameKey(testEntityName, "renée", nil), nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.PutString("renée", "bird")
		assert.NoError(t, err)
	}
}

func TestSiloPut(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Put", mock.Anything, newKeyWithNamespace("myLittleSilo", testEntityName, "renée"), mock.Anything).Return(newKeyWithNamespace("myLittleSilo", testEntityName, "renée"), nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.PutSiloString("myLittleSilo", "renée", "bird")
		assert.NoError(t, err)
	}
}

func TestReconnectOnPutFailure(t *testing.T) {
	// Very importantly, we set up our mock to *not* return an error on repeated calls for the same key
	mockDS := mockDatastore{returnNoErrOnRepeatedKey: true}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Put", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.PutString("renée", "bird")
		assert.NoError(t, err)
	}
}

func TestFailureToPutAfterReconnectOnFailure(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Put", mock.Anything, datastore.NameKey(testEntityName, "renée", nil), mock.Anything).Return(nil, fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.PutString("renée", "bird")
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}

func TestDeleteSuccessful(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Delete", mock.Anything, datastore.NameKey(testEntityName, "renée", nil)).Return(nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.DeleteString("renée")
		assert.NoError(t, err)
	}
}

func TestSiloDelete(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil)
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Delete", mock.Anything, newKeyWithNamespace("myLittleSilo", testEntityName, "renée")).Return(nil)

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.DeleteSiloString("myLittleSilo", "renée")
		assert.NoError(t, err)
	}
}

func TestReconnectOnDeleteFailure(t *testing.T) {
	// Very importantly, we set up our mock to *not* return an error on repeated calls for the same key
	mockDS := mockDatastore{returnNoErrOnRepeatedKey: true}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity)
	mockDS.On("Delete", mock.Anything, datastore.NameKey(testEntityName, "renée", nil)).Return(fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.DeleteString("renée")
		assert.NoError(t, err)
	}
}

func TestFailureToDeleteAfterReconnectOnFailure(t *testing.T) {
	mockDS := mockDatastore{}
	defer mockDS.AssertExpectations(t)

	mockDS.On("connect").Return(nil).Twice()
	mockDS.On("Get", mock.Anything, datastore.NameKey(testEntityName, "testConnectivity", nil), mock.Anything).Return(datastore.ErrNoSuchEntity).Twice()
	mockDS.On("Delete", mock.Anything, datastore.NameKey(testEntityName, "renée", nil)).Return(fmt.Errorf("rpc error: code = Unauthenticated"))

	dsdb, err := newWithDatastorer(testEntityName, &mockDS)
	assert.NoError(t, err)
	if assert.NotNil(t, dsdb) {
		err := dsdb.DeleteString("renée")
		if assert.Error(t, err) {
			assert.Equal(t, "rpc error: code = Unauthenticated", err.Error())
		}
	}
}
