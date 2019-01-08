// Package botservices provides services to bot accessible via slackscot and injected into Plugins
package botservices

import (
	"fmt"
	"github.com/alexandre-normand/slackscot/v2/slog"
	"github.com/hashicorp/golang-lru"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"
	"log"
)

// BotServices represents bot services made available to Plugins requiring the use of the slack api and/or more details on users (i.e. email or full name)
type BotServices struct {
	Client          *slack.Client
	UserInfoService *UserInfoService
	*log.Logger
}

// UserInfoService holds the cache and slackClient used to implement the user info functionality
type UserInfoService struct {
	slackClient      *slack.Client
	userProfileCache *lru.ARCCache
}

const (
	userInfoCacheSizeKey           = "userInfoCacheSize" // The number of entries to keep in the user info cache, int value. Defaults to no caching
	userInfoCacheSizeDisabledValue = 0
)

// userLoader is the function to load a value when not present in cache
type userLoader func(userId string) (up *slack.User, err error)

// New creates a new instance of BotServices with the provided
func New(v *viper.Viper, l *log.Logger, client *slack.Client) (botServices *BotServices, err error) {
	botServices = new(BotServices)
	botServices.Client = client
	if botServices.UserInfoService, err = newUserInfo(v, client); err != nil {
		return nil, err
	}

	botServices.Logger = l
	return botServices, nil
}

// newUserInfo creates a new user info bot service with caching if enabled via userProfileCacheSizeKey
func newUserInfo(v *viper.Viper, client *slack.Client) (userProfileService *UserInfoService, err error) {
	userProfileService = new(UserInfoService)

	s := v.GetInt(userInfoCacheSizeKey)

	if s > userInfoCacheSizeDisabledValue {
		userProfileService.userProfileCache, err = lru.NewARC(s)
		if err != nil {
			return nil, err
		}
	}

	userProfileService.slackClient = client

	return userProfileService, nil
}

// GetUserInfo gets the user info or returns an error and a zero value of slack.User is not found or
// an error occurred during retrieval
func (b *BotServices) GetUserInfo(userId string) (user slack.User, err error) {
	return b.getOrLoadUserInfo(userId, func(userId string) (usr *slack.User, err error) {
		return b.UserInfoService.slackClient.GetUserInfo(userId)
	})
}

// GetOrLoadUserInfo gets the user info from the cache (if used). If the entry isn't in cache, the info is loaded
// using the loader function and then added to the cache. If a user is not found or there's an error loading the entry
// using the loader function's execution, a zero value user info is returned along with the error
func (b *BotServices) getOrLoadUserInfo(userId string, loadUser userLoader) (userInfo slack.User, err error) {
	if b.UserInfoService.userProfileCache == nil {
		slog.Debugf(b.Logger, "Cache disabled, loading user info for [%s] from slack instead\n", userId)
		up, err := loadUser(userId)
		if err != nil {
			return slack.User{}, err
		}

		return *up, nil
	}

	if userProfile, exists := b.UserInfoService.userProfileCache.Get(userId); exists {
		slog.Debugf(b.Logger, "User info in cache [%s] so using that\n", userId)

		userProfile, ok := userProfile.(slack.User)
		if !ok {
			return slack.User{}, fmt.Errorf("Error converting cached value for user id [%s]: %v", userId, err)
		}

		return userProfile, nil
	}

	slog.Debugf(b.Logger, "User info for [%s] not found in cache, retrieving from slack and saving\n", userId)
	up, err := loadUser(userId)
	if err != nil {
		return slack.User{}, err
	}

	b.UserInfoService.userProfileCache.Add(userId, *up)

	return *up, nil
}
