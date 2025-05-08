package pay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"time"

	"github.com/gin-gonic/gin"

	"github.com/smartwalle/alipay/v3"
	"github.com/smartwalle/xid"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

var aliClient *alipay.Client

// 正式环境
const (
	rate = 7.3
)

func InitAlipay() {
	var err error
	ctx := context.Background()
	// 支付宝初始化
	if aliClient, err = alipay.New(os.Getenv("KAPP_ID"), os.Getenv("KPRIVATE_KEY"), true); err != nil {
		logger.Error(ctx, "初始化支付宝失败: "+err.Error())
		return
	}
	if err = aliClient.LoadAliPayPublicKey(os.Getenv("KPUBLIC_KEY")); err != nil {
		logger.Error(ctx, "加载支付宝公钥发生错误: "+err.Error())
		return
	}
	if err = aliClient.SetEncryptKey(os.Getenv("KENCRYPT_KEY")); err != nil {
		logger.Error(ctx, "加载内容加密密钥发生错误: "+err.Error())
		return
	}
}

// 预下单
func CreateAlipay(c *gin.Context) {
	var req CreateOrderRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "无效的参数",
			"success": false,
		})
		return
	}

	// 确认金额
	if req.Amount <= 0 || common.MultiplyFloatUnique(req.Amount, rate) != req.TotalAmount {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "金额异常",
		})
		return
	}

	// 创建订单号
	var tradeNo = fmt.Sprintf("%d", xid.Next())

	// 订单号关联用户
	order := model.OrderRecord{
		UserId:    c.GetInt(ctxkey.Id),
		TradeNo:   tradeNo,
		Quota:     int64(req.Amount * config.QuotaPerUnit),
		GrantType: 1,
		Status:    "WAIT_BUYER_PAY",
		CreateAt:  time.Now().Unix(),
		ExpiredAt: time.Now().Add(time.Minute * 15).Unix(), // 预下单有效时间为15分钟
	}
	err = order.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	domainURL := os.Getenv("SERVER_DOMAIN")
	// 创建支付宝订单
	var p = alipay.TradePreCreate{}
	p.Subject = "AiHubMix平台 API额度"
	p.OutTradeNo = tradeNo
	p.TotalAmount = strconv.FormatFloat(req.TotalAmount, 'f', -1, 64)
	p.ProductCode = "QR_CODE_OFFLINE"
	p.NotifyURL = domainURL + "/api/pay/alipay/notify"
	// 二维码有效期 2 小时
	res, err := aliClient.TradePreCreate(c, p)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 异步触发轮询
	val := DoneAsync(c, tradeNo)
	fmt.Println(<-val)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单信息",
		"data":    res,
	})
}

// 异步处理
func DoneAsync(c *gin.Context, tradeNo string) chan int {
	r := make(chan int)
	go func() {
		r <- 1
		// 3s后开启订单状态轮询
		time.Sleep(3 * time.Second)
		PollOrderStatus(c, tradeNo)
	}()
	return r
}

// 创建定时器 每隔 5s 查询支付宝订单状态 状态为成功时结束  2小时后自动结束定时器
func PollOrderStatus(c *gin.Context, tradeNo string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*15)
	defer cancel()

	for {
		select {
		// 超时取消订单
		case <-timeoutCtx.Done():
			CancelAlipayOrder(c, tradeNo)
			return
		// 定时查询订单状态
		case <-ticker.C:
			success, err := QueryAlipayOrder(c, tradeNo)
			if err != nil {
				logger.Error(context.Background(), "查询订单异常: "+err.Error())
				continue
			}
			if success {
				return
			}
		}
	}
}

// 查询支付宝订单状态
func QueryAlipayOrder(c *gin.Context, outTradeNo string) (bool, error) {
	var ctx = context.Background()
	var p = alipay.TradeQuery{}
	p.OutTradeNo = outTradeNo
	res, err := aliClient.TradeQuery(c, p)
	if err != nil {
		return false, err
	}

	// 交易不存在 生成二维码使用支付宝钱包扫码唤起收银台后 支付宝才会创建订单
	if res.Code == "40004" || res.TradeStatus == "" {
		return false, errors.New("订单不存在")
	}

	// 获取订单信息
	record, err := model.GetOrderByTradeNo(outTradeNo)
	if err != nil {
		return false, err
	}

	// 支付成功，更新用户额度
	if res.TradeStatus == "TRADE_SUCCESS" {
		// 订单状态为等待付款时更新用户额度
		if record.Status == "WAIT_BUYER_PAY" {
			// 更新用户额度
			err = model.IncreaseUserQuota(record.UserId, record.Quota)
			if err != nil {
				logger.Error(ctx, "更新用户额度异常: "+err.Error())
				return false, err
			}
			err = model.AddQuotaRecord(record.UserId, 1, record.TradeNo, record.Quota)
			if err != nil {
				logger.Error(ctx, "创建用户额度记录异常: "+err.Error())
				return false, err
			}

			// 添加额度变更记录
			model.RecordTopupLog(record.UserId, fmt.Sprintf("通过 支付宝 充值 %s", common.LogQuota(int64(record.Quota))), 0)
		}
	}

	// 更新订单状态
	err = model.UpdateOrderStatusByTradeNo(outTradeNo, string(res.TradeStatus))
	if err != nil {
		return false, err
	}

	// 是否订单状态是否变更，不是则返回false
	return string(res.TradeStatus) != record.Status, err
}

// 15分钟内未完成支付 关闭交易
func CancelAlipayOrder(c *gin.Context, outTradeNo string) {
	var p = alipay.TradeCancel{}
	p.OutTradeNo = outTradeNo
	res, err := aliClient.TradeCancel(c, p)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if res.Code != "10000" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": res.SubMsg,
		})
		return
	}

	err = model.UpdateOrderStatusByTradeNo(outTradeNo, "TRADE_CLOSED")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
	}
}

// 支付宝异步通知
func NotifyOrder(c *gin.Context) {
	var ctx = context.Background()
	c.Request.ParseForm()
	var notification, err = aliClient.DecodeNotification(c.Request.Form)
	if err != nil {
		logger.Error(ctx, "解析异步通知发生错误: "+err.Error())
		return
	}
	_, err = QueryAlipayOrder(c, notification.OutTradeNo)
	if err != nil {
		logger.Error(context.Background(), "支付宝异步通知 - 查询订单异常: "+err.Error())
	}

	aliClient.ACKNotification(c.Writer)
}
