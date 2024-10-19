package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	TokenCacheSeconds         = config.SyncFrequency
	UserId2GroupCacheSeconds  = config.SyncFrequency
	UserId2QuotaCacheSeconds  = config.SyncFrequency
	UserId2StatusCacheSeconds = config.SyncFrequency
	GroupModelsCacheSeconds   = config.SyncFrequency

	RecentChannelKeyPrefix = "recent_channel:%d:%s"
)

type Cache struct {
	Id       int64  `json:"id"`
	Key      string `json:"key" gorm:"type:varchar(255);uniqueIndex"`
	Value    string `json:"value"`
	ExpireAt int64  `json:"expire_at"`
}

func DeleteExpiredCache() {
	DB.Where("expire_at < ?", time.Now().Unix()).Delete(&Cache{})
}

func CacheGetRecentChannel(ctx context.Context, userId int, model string) (channelId int) {
	if config.IsZiai {
		return 0
	}
	key := fmt.Sprintf(RecentChannelKeyPrefix, userId, model)
	id, err := GetRecentChannelPool(key)
	if err == nil {
		return id
	}
	cache := &Cache{}
	err = DB.Where("`key` = ?", key).First(cache).Error
	if err != nil {
		return 0
	}
	if cache.ExpireAt < time.Now().Unix() {
		return 0
	}
	channelId, err = strconv.Atoi(cache.Value)
	if err != nil {
		logger.Error(ctx, "convert cache value to int error: "+err.Error())
		return 0
	}
	cache.ExpireAt = time.Now().Unix() + 3600
	_ = DB.Save(cache).Error
	SetRecentChannelPool(key, channelId)
	return channelId
}

func CacheSetRecentChannel(ctx context.Context, userId int, model string, channelId int) {
	if config.IsZiai {
		return
	}
	key := fmt.Sprintf(RecentChannelKeyPrefix, userId, model)
	id, err := GetRecentChannelPool(key)
	if err == nil && channelId == id {
		return
	}
	expireAt := time.Now().Unix() + 3600
	cache := &Cache{}
	err = DB.Where("`key` = ?", key).First(cache).Error
	if err != nil {
		cache = &Cache{
			Key:      key,
			Value:    strconv.Itoa(channelId),
			ExpireAt: expireAt,
		}
		err = DB.Create(cache).Error
		if err != nil {
			logger.Error(ctx, "create cache error: "+err.Error())
		}
		return
	}
	cache.Value = strconv.Itoa(channelId)
	cache.ExpireAt = expireAt
	err = DB.Save(cache).Error
	if err != nil {
		logger.Error(ctx, "save cache error: "+err.Error())
	} else {
		SetRecentChannelPool(key, channelId)
	}
}

func CacheGetTokenByKey(ctx context.Context, key string) (*Token, error) {
	ca := GetTokenByKeyPool(ctx, key)
	if ca != nil {
		return ca, nil
	}

	var token Token
	err := DB.Where("`key` = ?", key).First(&token).Error
	if err != nil {
		return nil, err
	}
	SetTokenPool(ctx, key, token)
	return &token, nil
}

func CacheGetUserGroup(ctx context.Context, id int) (group string, err error) {
	ca, err := GetUserGroupPool(ctx, id)
	if err == nil {
		return ca, nil
	}
	group, err = GetUserGroup(id)
	if err != nil {
		return "", err
	}
	SetUserGroupPool(ctx, id, group)
	return group, nil
}

func fetchAndUpdateUserQuota(ctx context.Context, id int) (quota int64, err error) {
	quota, err = GetUserQuota(id)
	if err != nil {
		return 0, err
	}
	err = common.RedisSet(fmt.Sprintf("user_quota:%d", id), fmt.Sprintf("%d", quota), time.Duration(UserId2QuotaCacheSeconds)*time.Second)
	if err != nil {
		logger.Error(ctx, "Redis set user quota error: "+err.Error())
	}
	return
}

func CacheGetUserQuota(ctx context.Context, id int) (quota int64, err error) {
	quota, err = GetUserQuotaPool(ctx, id)
	if err == nil {
		return quota, nil
	}

	quota, err = GetUserQuota(id)
	if err != nil {
		return 0, nil
	}
	SetUserQuotaPool(ctx, id, quota)
	return quota, nil
}

func CacheUpdateUserQuota(ctx context.Context, id int) error {
	if !common.RedisEnabled {
		return nil
	}
	quota, err := CacheGetUserQuota(ctx, id)
	if err != nil {
		return err
	}
	err = common.RedisSet(fmt.Sprintf("user_quota:%d", id), fmt.Sprintf("%d", quota), time.Duration(UserId2QuotaCacheSeconds)*time.Second)
	return err
}

func CacheDecreaseUserQuota(id int, quota int64) error {
	if !common.RedisEnabled {
		return nil
	}
	err := common.RedisDecrease(fmt.Sprintf("user_quota:%d", id), int64(quota))
	return err
}

func CacheIsUserEnabled(userId int) (bool, error) {
	if !common.RedisEnabled {
		return IsUserEnabled(userId)
	}
	enabled, err := common.RedisGet(fmt.Sprintf("user_enabled:%d", userId))
	if err == nil {
		return enabled == "1", nil
	}

	userEnabled, err := IsUserEnabled(userId)
	if err != nil {
		return false, err
	}
	enabled = "0"
	if userEnabled {
		enabled = "1"
	}
	err = common.RedisSet(fmt.Sprintf("user_enabled:%d", userId), enabled, time.Duration(UserId2StatusCacheSeconds)*time.Second)
	if err != nil {
		logger.SysError("Redis set user enabled error: " + err.Error())
	}
	return userEnabled, err
}

func CacheGetGroupModels(ctx context.Context, group string) ([]string, error) {
	if !common.RedisEnabled {
		return GetGroupModels(ctx, group)
	}
	modelsStr, err := common.RedisGet(fmt.Sprintf("group_models:%s", group))
	if err == nil {
		return strings.Split(modelsStr, ","), nil
	}
	models, err := GetGroupModels(ctx, group)
	if err != nil {
		return nil, err
	}
	err = common.RedisSet(fmt.Sprintf("group_models:%s", group), strings.Join(models, ","), time.Duration(GroupModelsCacheSeconds)*time.Second)
	if err != nil {
		logger.SysError("Redis set group models error: " + err.Error())
	}
	return models, nil
}

var group2model2channels map[string]map[string][]*Channel
var channelId2channel map[int]*Channel
var channelSyncLock sync.RWMutex

func InitChannelCache() {
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Where("status = ?", ChannelStatusEnabled).Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool)
	for _, ability := range abilities {
		groups[ability.Group] = true
	}
	newGroup2model2channels := make(map[string]map[string][]*Channel)
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]*Channel)
	}
	for _, channel := range channels {
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]*Channel, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel)
			}
		}
	}

	// sort by priority
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return channels[i].GetPriority() > channels[j].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	channelId2channel = newChannelId2channel
	channelSyncLock.Unlock()
	logger.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		logger.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func CacheGetChannelById(id int) (*Channel, error) {
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	channel, ok := channelId2channel[id]
	if !ok {
		return nil, errors.New("channel not found")
	}
	return channel, nil
}

func CacheGetRandomSatisfiedChannel(group string, model string, excludedChannelIds []int) (*Channel, error) {
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	if len(group2model2channels[group][model]) == 0 {
		return nil, errors.New("channel not found")
	}
	validChannels := make([]*Channel, 0)
	if excludedChannelIds == nil {
		excludedChannelIds = make([]int, 0)
	} else {
		for _, channel := range group2model2channels[group][model] {
			valid := true
			for _, excludedChannelId := range excludedChannelIds {
				if channel.Id == excludedChannelId {
					valid = false
					break
				}
			}
			if valid {
				validChannels = append(validChannels, channel)
			}
		}
	}
	if len(validChannels) == 0 {
		return nil, nil
	}
	endIdx := len(validChannels)
	// choose by priority
	firstChannel := validChannels[0]
	if firstChannel.GetPriority() > 0 {
		for i := range validChannels {
			if validChannels[i].GetPriority() != firstChannel.GetPriority() {
				endIdx = i
				break
			}
		}
	}
	idx := calcIdxByWeight(validChannels, endIdx)
	return validChannels[idx], nil
}

func calcIdxByWeight(channels []*Channel, endIdx int) int {
	if endIdx == 1 {
		return 0
	}
	totalWeight := 0
	for i, channel := range channels {
		if i < endIdx {
			totalWeight += getChannelWeight(channel)
		}
	}
	randomNum := rand.Intn(totalWeight)
	index := 0
	sum := 0
	for i, channel := range channels {
		sum += getChannelWeight(channel)
		if sum > randomNum {
			index = i
			break
		}
	}
	return index
}

func getChannelWeight(channel *Channel) int {
	if *channel.Weight <= 0 {
		return 1
	}
	return int(*channel.Weight)
}
