package model

import (
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"strconv"
	"sync"
	"time"
)

const (
	BatchUpdateTypeUserQuota = iota
	BatchUpdateTypeTokenQuota
	BatchUpdateTypeUsedQuota
	BatchUpdateTypeChannelUsedQuota
	BatchUpdateTypeRequestCount
	BatchUpdateTypeCount // if you add a new type, you need to add a new map and a new lock
)

var batchUpdateStores []map[int]int64
var batchUpdateLocks []sync.Mutex
var batchLogs []*Log
var batchLogsLock sync.Mutex

func init() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int64))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
	batchLogs = make([]*Log, 0)
}

func InitBatchUpdater() {
	go func() {
		for {
			time.Sleep(time.Duration(config.BatchUpdateInterval) * time.Second)
			batchUpdate()
			batchInsert()
		}
	}()
}

func addNewRecord(type_ int, id int, value int64) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	if _, ok := batchUpdateStores[type_][id]; !ok {
		batchUpdateStores[type_][id] = value
	} else {
		batchUpdateStores[type_][id] += value
	}
}

func addNewLog(log *Log) {
	batchLogsLock.Lock()
	defer batchLogsLock.Unlock()
	batchLogs = append(batchLogs, log)
}

func batchInsert() {
	logger.SysLog("batch insert started")
	batchLogsLock.Lock()
	logs := batchLogs
	batchLogs = make([]*Log, 0)
	batchLogsLock.Unlock()
	err := LOG_DB.CreateInBatches(logs, 100).Error

	location, _ := time.LoadLocation("Asia/Shanghai") // Beijing time zone
	startTime := time.Date(2024, 12, 1, 21, 10, 0, 0, location)
	if time.Now().Before(startTime) {
		return
	}
	usages := make(map[string]*Usage, 0)
	for _, log := range logs {
		key := strconv.Itoa(log.UserId) + log.ModelName + log.TokenName
		if _, ok := usages[key]; !ok {
			usages[key] = &Usage{
				UserId:       log.UserId,
				Hour:         getHour(),
				ModelName:    log.ModelName,
				TokenName:    log.TokenName,
				Count:        0,
				InputTokens:  0,
				OutputTokens: 0,
				Quota:        0,
			}
		}
		usage := usages[key]
		usage.Count++
		usage.InputTokens += log.PromptTokens
		usage.OutputTokens += log.CompletionTokens
		usage.Quota += log.Quota
	}
	for _, usage := range usages {
		err = AddUsage(usage.UserId, usage.ModelName, usage.TokenName, usage.Count, usage.InputTokens, usage.OutputTokens, usage.Quota)
		if err != nil {
			logger.SysError("failed to add usage: " + err.Error())
		}
	}

	if err != nil {
		logger.SysError("failed to batch insert logs: " + err.Error())
	}
}

func batchUpdate() {
	logger.SysLog("batch update started")
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		store := batchUpdateStores[i]
		batchUpdateStores[i] = make(map[int]int64)
		batchUpdateLocks[i].Unlock()
		// TODO: maybe we can combine updates with same key?
		for key, value := range store {
			switch i {
			case BatchUpdateTypeUserQuota:
				err := increaseUserQuota(key, value)
				if err != nil {
					logger.SysError("failed to batch update user quota: " + err.Error())
				}
			case BatchUpdateTypeTokenQuota:
				err := increaseTokenQuota(key, value)
				if err != nil {
					logger.SysError("failed to batch update token quota: " + err.Error())
				}
			case BatchUpdateTypeUsedQuota:
				updateUserUsedQuota(key, value)
			case BatchUpdateTypeRequestCount:
				updateUserRequestCount(key, int(value))
			case BatchUpdateTypeChannelUsedQuota:
				updateChannelUsedQuota(key, value)
			}
		}
	}
	logger.SysLog("batch update finished")
}
