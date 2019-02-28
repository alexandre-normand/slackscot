package inmemorydb_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/store/inmemorydb"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockStorer struct {
	data            map[string]string
	errorOnNextCall bool
	closed          bool
}

func newMockStorer(existingData map[string]string) (ms *mockStorer) {
	ms = new(mockStorer)
	ms.data = existingData
	return ms
}

func (ms *mockStorer) GetString(key string) (value string, err error) {
	if ms.errorOnNextCall {
		return "", fmt.Errorf("error with persistent db")
	}

	v, ok := ms.data[key]
	if !ok {
		return "", fmt.Errorf("%s not found", key)
	}

	return v, nil
}

func (ms *mockStorer) PutString(key string, value string) (err error) {
	if ms.errorOnNextCall {
		return fmt.Errorf("error with persistent db")
	}

	ms.data[key] = value
	return nil
}

func (ms *mockStorer) DeleteString(key string) (err error) {
	if ms.errorOnNextCall {
		return fmt.Errorf("error with persistent db")
	}

	delete(ms.data, key)
	return nil
}

func (ms *mockStorer) Scan() (entries map[string]string, err error) {
	if ms.errorOnNextCall {
		return nil, fmt.Errorf("error with persistent db")
	}

	entries = make(map[string]string)

	for k, v := range ms.data {
		entries[k] = v
	}

	return entries, nil
}

func (ms *mockStorer) Close() (err error) {
	if ms.errorOnNextCall {
		return fmt.Errorf("error with persistent db")
	}

	ms.closed = true
	return nil
}

func TestNewWithErrorLoadingPersistentContent(t *testing.T) {
	ms := &mockStorer{errorOnNextCall: true}

	_, err := inmemorydb.New(ms)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "error with persistent db")
	}
}

func TestGetWithPersistedExistingContent(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1", "key2": "value2"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		v1, err := imdb.GetString("key1")
		assert.Nil(t, err)
		assert.Equal(t, "value1", v1)

		v2, err := imdb.GetString("key2")
		assert.Nil(t, err)
		assert.Equal(t, "value2", v2)
	}
}

func TestScanExistingContent(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1", "key2": "value2"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		elements, err := imdb.Scan()
		// Modify the persistent storer to make sure that the map returned was a copy
		// and not the reference
		ms.data["key3"] = "should not be visible in the scan results"

		assert.Nil(t, err)
		assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, elements)
	}
}

func TestUpdateExistingContent(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1", "key2": "value2"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		err := imdb.PutString("key1", "bird")
		if assert.Nil(t, err) {
			imv1, err := imdb.GetString("key1")
			assert.Nil(t, err)
			assert.Equal(t, "bird", imv1)

			// Check it's also really persisted to the "persistent" storer
			msv1, err := ms.GetString("key1")
			assert.Nil(t, err)
			assert.Equal(t, "bird", msv1)
		}
	}
}

func TestDeleteExistingContent(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1", "key2": "value2"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		err := imdb.DeleteString("key1")
		if assert.Nil(t, err) {
			imv1, err := imdb.GetString("key1")
			assert.NotNil(t, err)
			assert.Equal(t, "", imv1)

			// Check it's also really deleted from the "persistent" storer
			msv1, err := ms.GetString("key1")
			assert.NotNil(t, err)
			assert.Equal(t, "", msv1)
		}
	}
}

func TestGetOnEmptyStorage(t *testing.T) {
	ms := &mockStorer{data: make(map[string]string)}

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		v1, err := imdb.GetString("key1")
		assert.Equal(t, "", v1)
		assert.EqualError(t, err, "key1 not found")
	}
}

func TestScanOnEmptyStorage(t *testing.T) {
	ms := &mockStorer{data: make(map[string]string)}

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		entries, err := imdb.Scan()
		assert.Nil(t, err)
		assert.Empty(t, entries)
	}
}

func TestCloseClosesPersistentStorage(t *testing.T) {
	ms := newMockStorer(map[string]string{})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		err := imdb.Close()

		assert.Nil(t, err)
		assert.Equalf(t, true, ms.closed, "Persistent db should be closed but wasn't")
	}
}

func TestErrorWithPersistentStorageOnGet(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		ms.errorOnNextCall = true

		_, err = ms.GetString("key1")
		// Confirm that the "persistent" storer errors out as instructed
		if assert.EqualError(t, err, "error with persistent db") {
			val, err := imdb.GetString("key1")
			// Validate that the in memory db doesn't interact with the persistent
			// storer on Get and returns what it has in memory
			assert.Nil(t, err)
			assert.Equal(t, "value1", val)
		}
	}
}

func TestErrorWithPersistentStorageOnScan(t *testing.T) {
	ms := newMockStorer(map[string]string{"key1": "value1"})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		ms.errorOnNextCall = true

		_, err = ms.Scan()
		// Confirm that the "persistent" storer errors out as instructed
		if assert.EqualError(t, err, "error with persistent db") {
			elements, err := imdb.Scan()
			// Validate that the in memory db doesn't interact with the persistent
			// storer on Scan and returns what it has in memory
			assert.Nil(t, err)
			assert.Equal(t, map[string]string{"key1": "value1"}, elements)
		}
	}
}

func TestErrorWithPersistentStorageOnPut(t *testing.T) {
	ms := newMockStorer(map[string]string{})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		ms.errorOnNextCall = true

		err := imdb.PutString("key1", "value1")

		assert.EqualError(t, err, "error with persistent db")
	}
}

func TestErrorWithPersistentStorageOnDelete(t *testing.T) {
	ms := newMockStorer(map[string]string{})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		ms.errorOnNextCall = true

		err := imdb.DeleteString("key1")

		assert.EqualError(t, err, "error with persistent db")
	}
}

func TestErrorClosingPersistentStorerIsReturned(t *testing.T) {
	ms := newMockStorer(map[string]string{})

	imdb, err := inmemorydb.New(ms)
	if assert.Nil(t, err) {
		ms.errorOnNextCall = true

		err := imdb.Close()

		assert.EqualError(t, err, "error with persistent db")
	}
}
