package model

import (
	"context"
	"encoding/json"
	"github.com/coocood/freecache"
	"github.com/songquanpeng/one-api/common/logger"
	"strconv"
)

var TokenKeyCache *freecache.Cache
var TokenIdCache *freecache.Cache
var UserGroupCache *freecache.Cache
var UserQuotaCache *freecache.Cache
var UsernamesCache *freecache.Cache
var RecentChannelCache *freecache.Cache

func InitPool() {
	TokenKeyCache = freecache.NewCache(10 * 1024 * 1024)
	TokenIdCache = freecache.NewCache(10 * 1024 * 1024)
	UserGroupCache = freecache.NewCache(0)
	UserQuotaCache = freecache.NewCache(0)
	UsernamesCache = freecache.NewCache(0)
	RecentChannelCache = freecache.NewCache(1 * 1024 * 1024)
}

// GetTokenByKey gets token by key
func GetTokenByKeyPool(ctx context.Context, key string) *Token {
	tokenBytes, err := TokenKeyCache.Get([]byte(key))
	if err != nil {
		return nil
	}
	var token Token
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		logger.Error(ctx, "unmarshal token error: "+err.Error())
		return nil
	}
	return &token
}

// SetToken sets token
func SetTokenPool(ctx context.Context, key string, token Token) {
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		logger.Error(ctx, "marshal token error: "+err.Error())
		return
	}
	err = TokenKeyCache.Set([]byte(key), tokenBytes, 60)
	if err != nil {
		logger.Error(ctx, "set token error: "+err.Error())
	}
}

// GetTokenById gets token by id
func GetTokenByIdPool(id int) *Token {
	tokenBytes, err := TokenIdCache.Get([]byte(strconv.Itoa(id)))
	if err != nil {
		return nil
	}
	var token Token
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		logger.SysError("unmarshal token error: " + err.Error())
		return nil
	}
	return &token
}

// SetToken sets token
func SetTokenByIdPool(id int, token Token) {
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		logger.SysError("marshal token error: " + err.Error())
		return
	}
	err = TokenIdCache.Set([]byte(strconv.Itoa(id)), tokenBytes, 60)
	if err != nil {
		logger.SysError("set token error: " + err.Error())
	}
}

// GetUserGroup gets user group
func GetUserGroupPool(ctx context.Context, userId int) (string, error) {
	groupBytes, err := UserGroupCache.Get([]byte(strconv.Itoa(userId)))
	if err != nil {
		return "", err
	}
	return string(groupBytes), nil
}

// SetUserGroup sets user group
func SetUserGroupPool(ctx context.Context, userId int, group string) {
	err := UserGroupCache.Set([]byte(strconv.Itoa(userId)), []byte(group), 60)
	if err != nil {
		logger.Error(ctx, "set user group error: "+err.Error())
	}
}

// GetUserQuota gets user quota
func GetUserQuotaPool(ctx context.Context, userId int) (int64, error) {
	quotaBytes, err := UserQuotaCache.Get([]byte(strconv.Itoa(userId)))
	if err != nil {
		return 0, err
	}
	quota, err := strconv.ParseInt(string(quotaBytes), 10, 64)
	if err != nil {
		logger.Error(ctx, "parse user quota error: "+err.Error())
		return 0, err
	}
	return quota, nil
}

// SetUserQuota sets user quota
func SetUserQuotaPool(ctx context.Context, userId int, quota int64) {
	err := UserQuotaCache.Set([]byte(strconv.Itoa(userId)), []byte(strconv.FormatInt(quota, 10)), 10)
	if err != nil {
		logger.Error(ctx, "set user quota error: "+err.Error())
	}
}

// GetUsernames gets usernames
func GetUsernamePool(userId int) (string, error) {
	usernameBytes, err := UsernamesCache.Get([]byte(strconv.Itoa(userId)))
	if err != nil {
		return "", err
	}
	return string(usernameBytes), nil
}

// SetUsernames sets usernames
func SetUsernamePool(userId int, username string) {
	err := UsernamesCache.Set([]byte(strconv.Itoa(userId)), []byte(username), 60)
	if err != nil {
		logger.SysError("set usernames error: " + err.Error())
	}
}

// GetRecentChannel gets recent channel
func GetRecentChannelPool(key string) (int, error) {
	channelBytes, err := RecentChannelCache.Get([]byte(key))
	if err != nil {
		return 0, err
	}
	channelId, err := strconv.Atoi(string(channelBytes))
	if err != nil {
		logger.SysError("parse recent channel error: " + err.Error())
		return 0, err
	}
	return channelId, nil
}

// SetRecentChannel sets recent channel
func SetRecentChannelPool(key string, channelId int) {
	err := RecentChannelCache.Set([]byte(key), []byte(strconv.Itoa(channelId)), 60)
	if err != nil {
		logger.SysError("set recent channel error: " + err.Error())
	}
}
