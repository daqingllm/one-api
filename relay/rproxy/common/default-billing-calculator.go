package common

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type ChargeMode int

const (
	// TokenUsage 表示按Token使用量计费
	TokenUsage ChargeMode = iota
	// PayPerUse 表示按次计费
	PayPerUse
)

type ItemType int

const (
	//
	PromptTokens ItemType = iota
	CompletionTokens
	CachedTokens
	CachedStorage
	ToolUsePromoptTokens
	ThoughtsTokens
	WebSearch
	ImageToken
	VideoToken
)

func (i ItemType) String() string {
	names := []string{
		"PromptTokens",
		"CompletionTokens",
		"CachedTokens",
		"CachedStorage",
		"ToolUsePromoptTokens",
		"ThoughtsTokens",
		"WebSearch",
		"ImageToken",
		"VideoToken",
	}

	if int(i) < 0 || int(i) >= len(names) {
		return fmt.Sprintf("Unknown(%d)", i)
	}

	return names[int(i)]
}

type BillItem struct {
	ID            int64
	Name          string
	ItemType      ItemType
	ChargeMode    ChargeMode
	UnitPrice     float64
	Quantity      float64
	Discount      *Discount
	DiscountQuota int64
	Quota         int64
	Cost          float64
}

type DiscountType int

type Discount struct {
	ID       string
	Name     string
	Type     DiscountType
	Ratio    float64
	Describe string
}

type Bill struct {
	BillID           int64
	ChannelId        int
	ChannelName      string
	ModelName        string
	PreBillItems     []*BillItem
	BillItems        []*BillItem
	PreOriginalQuota int64
	PreTotalQuota    int64
	PreDiscountQuota int64
	OriginalQuota    int64
	DiscountQuota    int64
	TotalQuota       int64
	Discounts        []*Discount
	Extra            map[string]any
}

type DefaultBillingCalculator struct {
	adaptor               rproxy.RproxyAdaptor
	Bill                  *Bill
	PreCalcStrategyFunc   func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
	CalcStrategyFunc      func(context *rproxy.RproxyContext, channel *model.Channel, groupRatio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode)
	PostCalcStrategyFunc  func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
	FinalCalcStrategyFunc func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
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
	b.Bill = &Bill{
		ChannelId:    channel.Id,
		ChannelName:  channel.Name,
		ModelName:    context.GetOriginalModel(),
		PreBillItems: make([]*BillItem, 0),
		BillItems:    make([]*BillItem, 0),
		Discounts:    make([]*Discount, 0),
	}
	// 获取并添加模型倍率折扣
	modelRatio := ratio.GetModelRatio(context.GetOriginalModel(), channel.Type)
	b.Bill.Discounts = append(b.Bill.Discounts, &Discount{
		ID:       "model_ratio",
		Name:     "模型倍率",
		Type:     0, // 0 表示模型级折扣
		Ratio:    modelRatio,
		Describe: fmt.Sprintf("模型 %s 费率系数", context.GetOriginalModel()),
	})

	// 获取并添加分组倍率折扣
	groupRatio := ratio.GetGroupRatio(context.Meta.Group)
	b.Bill.Discounts = append(b.Bill.Discounts, &Discount{
		ID:       "group_ratio",
		Name:     "分组倍率",
		Type:     1, // 1 表示分组级折扣
		Ratio:    groupRatio,
		Describe: fmt.Sprintf("用户组 %s 折扣系数", context.Meta.Group),
	})

	if b.PreCalcStrategyFunc != nil {
		e := b.PreCalcStrategyFunc(context, channel, b.Bill)
		if e != nil {
			return e
		}
	}
	b.calcPreTotalBill()

	userQuota, err := model.CacheGetUserQuota(context.SrcContext, context.GetUserId())
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota-b.Bill.PreTotalQuota < 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}
	err = model.CacheDecreaseUserQuota(context.GetUserId(), b.Bill.PreTotalQuota)
	if err != nil {
		return openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	e := model.PreConsumeTokenQuota(context.Meta.TokenId, b.Bill.PreTotalQuota)
	if e != nil {
		logger.Error(context.SrcContext, "error return pre-consumed quota: "+e.Error())
		return openai.ErrorWrapper(e, "decrease_user_quota_failed", http.StatusInternalServerError)
	}
	return nil
}
func (b *DefaultBillingCalculator) RollBackPreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	if b.Bill.PreTotalQuota > 0 {
		go func(ctx *rproxy.RproxyContext, preConsumedQuota int64) {
			err := model.PostConsumeTokenQuota(ctx.Meta.TokenId, -preConsumedQuota)
			if err != nil {
				logger.Error(ctx.SrcContext, "error return pre-consumed quota: "+err.Error())
			}
		}(context, b.Bill.PreTotalQuota)
	}
	return nil
}
func (b *DefaultBillingCalculator) PostCalcAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	//todo add post-consumed quota
	if b.PostCalcStrategyFunc != nil {
		b.PostCalcStrategyFunc(context, b.GetChannel(), b.Bill)
	}
	//计算总费率
	b.calcTotalBill()
	if b.Bill.TotalQuota <= 0 {
		return nil
	}
	if config.DebugUserIds[context.GetUserId()] {
		logger.DebugForcef(context.SrcContext, "usage:%v", b.Bill)
	}
	var logContent string
	for _, discount := range b.Bill.Discounts {
		logContent += fmt.Sprintf("%s %.3f，", discount.Name, discount.Ratio)
	}
	for _, item := range b.Bill.BillItems {
		if item.Discount != nil {
			logContent += fmt.Sprintf("%s %.3f，", item.Discount.Name, item.Discount.Ratio)
		}
		if item.ChargeMode == PayPerUse {
			logContent += fmt.Sprintf("%s费用 %4f，", item.ItemType.String(), item.Cost)
		}
	}
	var promptTokens int = 0
	var completionTokens int = 0
	var cachedTokens int = 0
	for _, billItem := range b.Bill.BillItems {
		if PayPerUse == billItem.ChargeMode {
			continue
		}
		switch billItem.Name {
		case "PromptTokens":
			promptTokens += int(billItem.Quantity)
		case "CompletionTokens":
			completionTokens += int(billItem.Quantity)
		case "CachedTokens":
			cachedTokens += int(billItem.Quantity)
		default:
			promptTokens += int(billItem.Quantity)
		}
	}
	err := model.PostConsumeTokenQuota(context.Meta.TokenId, b.Bill.TotalQuota-b.Bill.PreTotalQuota)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	err = model.CacheUpdateUserQuota(context.SrcContext, context.GetUserId())
	if err != nil {
		logger.SysError("error update user quota cache: " + err.Error())
	}
	model.RecordConsumeLog(context.SrcContext, context.GetUserId(), b.GetChannel().Id, promptTokens, cachedTokens, completionTokens, b.Bill.ModelName, context.Meta.TokenName, b.Bill.TotalQuota, logContent)
	model.UpdateUserUsedQuotaAndRequestCount(context.GetUserId(), b.Bill.TotalQuota)
	model.UpdateChannelUsedQuota(b.GetChannel().Id, b.Bill.TotalQuota)
	return nil
}

func (b *DefaultBillingCalculator) calcPreTotalBill() {
	if b.Bill == nil {
		return
	}
	totalPreOriginal := int64(0)

	for _, item := range b.Bill.PreBillItems {
		totalPreOriginal += item.Quota
	}

	var ratio float64 = 1
	for _, discount := range b.Bill.Discounts {
		ratio = ratio * discount.Ratio
	}
	b.Bill.PreOriginalQuota = totalPreOriginal
	b.Bill.PreDiscountQuota = int64(float64(totalPreOriginal) * ratio)
	b.Bill.PreTotalQuota = b.Bill.PreDiscountQuota
}
func (b *DefaultBillingCalculator) calcTotalBill() {
	if b.Bill == nil {
		return
	}
	if len(b.Bill.BillItems) == 0 {
		b.Bill.BillItems = b.Bill.PreBillItems
	} else {
		for _, item := range b.Bill.PreBillItems {
			if item.ChargeMode == PayPerUse {
				b.Bill.BillItems = append(b.Bill.BillItems, item)
			}
		}
	}
	totalOriginal := int64(0)
	var payPerUseQuota int64 = 0
	for _, item := range b.Bill.BillItems {
		switch item.ChargeMode {
		case PayPerUse:
			payPerUseQuota += item.Quota
		default:
			totalOriginal += item.Quota
		}
	}
	var ratio float64 = 1
	for _, discount := range b.Bill.Discounts {
		ratio = ratio * discount.Ratio

	}
	b.Bill.OriginalQuota = totalOriginal
	b.Bill.DiscountQuota = int64(float64(totalOriginal) * ratio)
	b.Bill.TotalQuota = b.Bill.DiscountQuota + payPerUseQuota
}

func PayperUseBillItem(itemType ItemType, unitPrice float64, quantity float64) *BillItem {
	return &BillItem{
		ChargeMode: PayPerUse,
		ItemType:   itemType,
		UnitPrice:  unitPrice,
		Quantity:   quantity,
		Quota:      int64(unitPrice * ratio.USD * 1000 * quantity),
		Cost:       unitPrice * quantity,
	}
}

func TokenUsageBillItem(itemType ItemType, unitPrice float64, quantity float64) *BillItem {
	return &BillItem{
		ChargeMode: TokenUsage,
		ItemType:   itemType,
		UnitPrice:  unitPrice,
		Quantity:   quantity,
		Quota:      int64(unitPrice * quantity),
	}
}
