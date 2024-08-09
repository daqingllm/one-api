package job

import (
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"testing"
	"time"
)

func TestCalcModelUsageSchedule(t *testing.T) {
	var err error
	model.DB, err = model.InitDB("SQL_DSN")
	if err != nil {
		logger.FatalLog("failed to initialize database: " + err.Error())
	}
	CalcModelUsageSchedule()
	time.Sleep(time.Minute * 30)
}
