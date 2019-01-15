package store_test

import (
	"github.com/alexandre-normand/slackscot/v2/store"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewStoreWithInvalidPath(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	assert.Nil(t, err)

	defer os.Remove(tmpfile.Name()) // clean up

	_, err = store.New("test", tmpfile.Name())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "failed to open")
	}
}

func TestNewStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)

	defer os.RemoveAll(dir)

	ts, err := store.New("test", dir)
	assert.Nil(t, err)

	assert.Equal(t, "test", ts.Name)
}

func TestPutGetScan(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmpTest")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	ts, err := store.New("test", dir)
	assert.Nil(t, err)
	assert.Equal(t, "test", ts.Name)

	err = ts.Put("testKey", "value1")
	assert.Nil(t, err)

	v, err := ts.Get("testKey")
	assert.Nil(t, err)

	assert.Equal(t, "value1", v)

	m, err := ts.Scan()
	assert.Nil(t, err)

	assert.Equal(t, map[string]string{"testKey": "value1"}, m)
}
