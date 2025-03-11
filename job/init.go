package job

func Init() {
	CalcModelUsageSchedule()
	ExpireCache()
	QuotaJob()
	ExpireHistoryLogs()
}
