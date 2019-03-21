package store_test

import (
	"github.com/alexandre-normand/slackscot/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	_, err = bs.GetString("testKey")
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

	_, err = bs.Get([]byte("testKey"))
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

func TestPutGetScanSiloString(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var sstorer store.SiloStringStorer

	sstorer, err = store.NewLevelDB("test", dir)
	assert.NoError(t, err)
	defer sstorer.Close()

	err = sstorer.PutSiloString("ns1", "testKey", "value1")
	assert.NoError(t, err)

	_, err = sstorer.GetSiloString("otherns1", "testKey")
	assert.Error(t, err)

	v, err := sstorer.GetSiloString("ns1", "testKey")
	assert.NoError(t, err)

	assert.Equal(t, "value1", v)

	m, err := sstorer.ScanSilo("ns1")
	assert.NoError(t, err)

	assert.Equal(t, map[string]string{"testKey": "value1"}, m)
}

func TestGlobalScan(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var sstorer store.GlobalSiloStringStorer

	sstorer, err = store.NewLevelDB("test", dir)
	assert.NoError(t, err)
	defer sstorer.Close()

	err = sstorer.PutSiloString("ns1", "testKey", "value1")
	require.NoError(t, err)

	err = sstorer.PutSiloString("ns2", "testKey2", "value2")
	require.NoError(t, err)

	err = sstorer.PutSiloString("", "testKey", "value2")
	require.NoError(t, err)

	m, err := sstorer.GlobalScan()
	require.NoError(t, err)

	assert.Equal(t, map[string]map[string]string{"ns1": map[string]string{"testKey": "value1"}, "ns2": map[string]string{"testKey2": "value2"}, "": map[string]string{"testKey": "value2"}}, m)
}
