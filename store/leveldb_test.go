package store_test

import (
	"github.com/alexandre-normand/slackscot/store"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewStoreWithInvalidPath(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	assert.Nil(t, err)

	defer os.Remove(tmpfile.Name()) // clean up

	_, err = store.NewLevelDB("test", tmpfile.Name())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to open")
	}
}

func TestNewLevelDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)

	defer os.RemoveAll(dir)

	ldb, err := store.NewLevelDB("test", dir)
	assert.Nil(t, err)
	defer ldb.Close()

	assert.Equal(t, "test", ldb.Name)
}

func TestGetAfterCloseShouldResultInError(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)

	defer os.RemoveAll(dir)

	ldb, err := store.NewLevelDB("test", dir)
	assert.Nil(t, err)

	ldb.Close()
	_, err = ldb.Get([]byte("testKey"))

	assert.Error(t, err)
}

func TestPutGetScanAsBytes(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var bs store.BytesStorer

	bs, err = store.NewLevelDB("test", dir)
	assert.Nil(t, err)
	defer bs.Close()

	err = bs.Put([]byte("testKey"), []byte("value1"))
	assert.Nil(t, err)

	v, err := bs.Get([]byte("testKey"))
	assert.Nil(t, err)

	assert.Equal(t, []byte("value1"), v)

	m, err := bs.Scan()
	assert.Nil(t, err)

	assert.Equal(t, map[string]string{"testKey": "value1"}, m)
}

func TestDeleteString(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var bs store.StringStorer

	bs, err = store.NewLevelDB("test", dir)
	assert.Nil(t, err)
	defer bs.Close()

	err = bs.PutString("testKey", "value1")
	assert.Nil(t, err)

	v, err := bs.GetString("testKey")
	assert.Nil(t, err)

	assert.Equal(t, "value1", v)

	err = bs.DeleteString("testKey")
	assert.Nil(t, err)

	v, err = bs.GetString("testKey")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "not found")
	}
}

func TestDeleteAsBytes(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var bs store.BytesStorer

	bs, err = store.NewLevelDB("test", dir)
	assert.Nil(t, err)
	defer bs.Close()

	err = bs.Put([]byte("testKey"), []byte("value1"))
	assert.Nil(t, err)

	v, err := bs.Get([]byte("testKey"))
	assert.Nil(t, err)

	assert.Equal(t, []byte("value1"), v)

	err = bs.Delete([]byte("testKey"))
	assert.Nil(t, err)

	v, err = bs.Get([]byte("testKey"))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "not found")
	}
}

func TestPutGetScanAsString(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var sstorer store.StringStorer

	sstorer, err = store.NewLevelDB("test", dir)
	assert.Nil(t, err)
	defer sstorer.Close()

	err = sstorer.PutString("testKey", "value1")
	assert.Nil(t, err)

	v, err := sstorer.GetString("testKey")
	assert.Nil(t, err)

	assert.Equal(t, "value1", v)

	m, err := sstorer.Scan()
	assert.Nil(t, err)

	assert.Equal(t, map[string]string{"testKey": "value1"}, m)
}
