package slackscot_test

import (
	"fmt"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"testing"
)

type userInfoFinder struct {
	fail bool
}

func (u *userInfoFinder) GetUserInfo(userID string) (user *slack.User, err error) {
	if u.fail {
		return nil, fmt.Errorf("Error loading user [%s]", userID)
	}

	return &slack.User{ID: userID, Name: "Daniel Quinn"}, nil
}

func NewCachingUserInfoFinderWithInvalidSize(t *testing.T) {
	v := viper.New()
	v.Set(config.UserInfoCacheSizeKey, -1)

	userInfoFinder := userInfoFinder{}
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)
	sLogger := slackscot.NewSLogger(logger, false)

	_, err := slackscot.NewCachingUserInfoFinder(v, &userInfoFinder, sLogger)
	assert.NotNil(t, err)
}

func NewGetUserWithCacheDisabled(t *testing.T) {
	v := viper.New()
	v.Set(config.UserInfoCacheSizeKey, 0)

	userInfoFinder := userInfoFinder{}
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)
	sLogger := slackscot.NewSLogger(logger, false)

	uf, err := slackscot.NewCachingUserInfoFinder(v, &userInfoFinder, sLogger)
	if assert.Nil(t, err) {
		user, err := uf.GetUserInfo("little-blue")
		assert.Nil(t, err)

		if assert.NotNil(t, user) {
			assert.Equal(t, slack.User{ID: "little-blue", Name: "Daniel Quinn"}, *user)
		}
	}
}

func NewGetUserFailToLoad(t *testing.T) {
	v := viper.New()
	v.Set(config.UserInfoCacheSizeKey, 1)

	userInfoFinder := userInfoFinder{}
	var logBuilder strings.Builder
	logger := log.New(&logBuilder, "", 0)
	sLogger := slackscot.NewSLogger(logger, false)

	uf, err := slackscot.NewCachingUserInfoFinder(v, &userInfoFinder, sLogger)
	if assert.Nil(t, err) {
		_, err := uf.GetUserInfo("little-blue")
		assert.NotNil(t, err)
	}
}
