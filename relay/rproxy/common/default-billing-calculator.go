package common

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type DefaultBillingCalculator struct {
	groupRatio       float64
	ratio            float64
	adaptor          rproxy.RproxyAdaptor
	preConsumedQuota int64
	CalcStrategyFunc func(context *rproxy.RproxyContext, channel *model.Channel, groupRatio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode)
}

func (b *DefaultBillingCalculator) GetChannel() *model.Channel {
	if b.adaptor == nil {
		return nil
	}
	if channel := b.adaptor.GetChannel(); channel != nil {
		return channel
	}
	return nil
}
func (b *DefaultBillingCalculator) PreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	var channel = b.GetChannel()
	if channel == nil {
		return openai.ErrorWrapper(errors.New("channel is nil"), "channel_is_nil", http.StatusInternalServerError)

	}
	// b.modelRatio = ratio.GetModelRatio(context.GetOriginalModel(), channel.Type)
	b.groupRatio = ratio.GetGroupRatio(context.Meta.Group)
	// b.ratio = b.modelRatio * b.groupRatio
	if b.CalcStrategyFunc != nil {
		quota, e := b.CalcStrategyFunc(context, channel, b.groupRatio)
		if e != nil {
			return e
		}
		b.preConsumedQuota = quota
	}

	userQuota, err := model.CacheGetUserQuota(context.SrcContext, context.GetUserId())
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-b.preConsumedQuota < 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(context.GetUserId(), b.preConsumedQuota)
	if err != nil {
		return openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	return nil
}
func (b *DefaultBillingCalculator) RollBackPreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	if b.preConsumedQuota > 0 {
		go func(ctx *rproxy.RproxyContext, preConsumedQuota int64) {
			err := model.PostConsumeTokenQuota(ctx.Meta.TokenId, -preConsumedQuota)
			if err != nil {
				logger.Error(ctx.SrcContext, "error return pre-consumed quota: "+err.Error())
			}

		}(context, b.preConsumedQuota)
	}
	return nil
}
func (b *DefaultBillingCalculator) PostCalcAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	//todo add post-consumed quota
	if b.preConsumedQuota <= 0 {
		return nil
	}
	logContent := fmt.Sprintf("分组倍率 %.3f，", b.groupRatio)
	model.RecordConsumeLog(context.SrcContext, context.GetUserId(), b.GetChannel().Id, int(b.preConsumedQuota), 0, 0, context.Meta.OriginModelName, context.Meta.TokenName, b.preConsumedQuota, logContent)
	model.UpdateChannelUsedQuota(b.GetChannel().Id, b.preConsumedQuota)
	return nil
}
