package slackscot

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/config"
	"github.com/hashicorp/golang-lru"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

const (
	userInfoCacheSizeDisabledValue = 0
)

// UserInfoFinder defines the interface for finding a slack user's info
type UserInfoFinder interface {
	GetUserInfo(userID string) (user *slack.User, err error)
}

// selfInfoFinder defines the interface for finding our (the slackscot instance) user info
type selfInfoFinder interface {
	GetInfo() (user *slack.Info)
}

// cachingUserInfoFinder holds a cache and a loading UserInfoFinder to implement the UserInfoFinder loading entries from cache
type cachingUserInfoFinder struct {
	loader           UserInfoFinder
	logger           SLogger
	userProfileCache *lru.ARCCache
}

// NewCachingUserInfoFinder creates a new user info service with caching if enabled via userProfileCacheSizeKey. It requires an implementation
// of the interface that will do the actual loading when not in cache
func NewCachingUserInfoFinder(v *viper.Viper, loader UserInfoFinder, logger SLogger) (uf UserInfoFinder, err error) {
	cuf := new(cachingUserInfoFinder)

	cs := v.GetInt(config.UserInfoCacheSizeKey)

	if cs > userInfoCacheSizeDisabledValue {
		cuf.userProfileCache, err = lru.NewARC(cs)
		if err != nil {
			return nil, err
		}
	}

	cuf.loader = loader
	cuf.logger = logger

	return cuf, nil
}

// GetUserInfo gets the user info or returns an error and a nil user is not found or
// an error occurred during retrieval
func (c cachingUserInfoFinder) GetUserInfo(userID string) (u *slack.User, err error) {
	if c.userProfileCache == nil {
		c.logger.Debugf("Cache disabled, loading user info for [%s] from slack instead\n", userID)
		u, err := c.loader.GetUserInfo(userID)
		if err != nil {
			return nil, err
		}

		return u, nil
	}

	if userProfile, exists := c.userProfileCache.Get(userID); exists {
		c.logger.Debugf("User info in cache [%s] so using that\n", userID)

		userProfile, ok := userProfile.(slack.User)
		if !ok {
			return nil, fmt.Errorf("Error converting cached value for user id [%s]: %v", userID, err)
		}

		return &userProfile, nil
	}

	c.logger.Debugf("User info for [%s] not found in cache, retrieving from slack and saving\n", userID)
	u, err = c.loader.GetUserInfo(userID)
	if err != nil {
		return nil, err
	}

	c.userProfileCache.Add(userID, *u)

	return u, nil
}
