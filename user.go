package slackscot

import (
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

const (
	userInfoCacheSizeKey           = "userInfoCacheSize" // The number of entries to keep in the user info cache, int value. Defaults to no caching
	userInfoCacheSizeDisabledValue = 0
)

// UserInfoFinder defines the interface for finding a slack user's info
type UserInfoFinder interface {
	GetUserInfo(userId string) (user *slack.User, err error)
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

	cs := v.GetInt(userInfoCacheSizeKey)

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
func (c cachingUserInfoFinder) GetUserInfo(userId string) (u *slack.User, err error) {
	if c.userProfileCache == nil {
		c.logger.Debugf("Cache disabled, loading user info for [%s] from slack instead\n", userId)
		u, err := c.loader.GetUserInfo(userId)
		if err != nil {
			return nil, err
		}

		return u, nil
	}

	if userProfile, exists := c.userProfileCache.Get(userId); exists {
		c.logger.Debugf("User info in cache [%s] so using that\n", userId)

		userProfile, ok := userProfile.(slack.User)
		if !ok {
			return nil, fmt.Errorf("Error converting cached value for user id [%s]: %v", userId, err)
		}

		return &userProfile, nil
	}

	c.logger.Debugf("User info for [%s] not found in cache, retrieving from slack and saving\n", userId)
	u, err = c.loader.GetUserInfo(userId)
	if err != nil {
		return nil, err
	}

	c.userProfileCache.Add(userId, *u)

	return u, nil
}
