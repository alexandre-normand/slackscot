package plugins

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	maxGifSizeInBytes = 10000000
)

// TestPicturesSmallerThan10MB checks that all urls are for GIFs smaller than 10MB
// The main use is for contributions to the curated list to be confirmed by PR builds
// to be of acceptable size
func TestPicturesSmallerThan10MB(t *testing.T) {
	for _, url := range mondayPictures {
		t.Run(url, func(t *testing.T) {
			size, err := getGifSize(url)
			assert.Nil(t, err)

			assert.Conditionf(t, func() bool { return size <= maxGifSizeInBytes }, "Gif file size should be <= %d bytes but was %d bytes for [%s]", maxGifSizeInBytes, size, url)
		})
	}
}

func getGifSize(gifURL string) (sizeInBytes int, err error) {
	resp, err := http.Get(gifURL)
	if err != nil {
		return 0, errors.Wrapf(err, "Error loading gif url [%s]", gifURL)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	return len(b), err
}
