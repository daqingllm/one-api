package pay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

func InitStripe() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

func CreateStripe(c *gin.Context) {
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

	domainURL := os.Getenv("SERVER_DOMAIN")

	// 创建stripe订单
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(os.Getenv("PRICE_ID")),
				Quantity: stripe.Int64(int64(req.Amount)),
			},
		},
		Mode:         stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:   stripe.String(domainURL + "/api/pay/stripe/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:    stripe.String(domainURL + "/api/pay/stripe/failed?session_id={CHECKOUT_SESSION_ID}"),
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
	}
	s, err := session.New(params)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 订单号关联用户
	order := model.OrderRecord{
		UserId:    c.GetInt(ctxkey.Id),
		TradeNo:   s.ID,
		Quota:     int64(req.Amount * config.QuotaPerUnit),
		GrantType: 2,
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建订单成功",
		"data":    s.URL,
	})
}

// Stripe订单支付成功
func StripeOrderSuccess(c *gin.Context) {
	sessionId := c.Query("session_id")
	res, err := session.Get(sessionId, nil)
	if err != nil {
		logger.Error(c, "获取stripe支付信息异常: "+err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 获取订单信息
	record, err := model.GetOrderByTradeNo(sessionId)
	if err != nil {
		logger.Error(c, "获取订单信息异常: "+err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	tradeStatus := record.Status

	// 支付成功，更新订单状态
	if res.PaymentStatus == "paid" {
		// 订单状态为等待付款时更新用户额度
		if tradeStatus == "WAIT_BUYER_PAY" {
			// 更新用户额度
			err = model.UpdateUserQuota(record.UserId, record.Quota)
			if err != nil {
				logger.Error(c, "更新用户额度异常: "+err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "更新用户额度异常",
				})
				return
			}
			err = model.AddQuotaRecord(record.UserId, 2, record.TradeNo, record.Quota)
			if err != nil {
				logger.Error(c, "创建用户额度记录异常: "+err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "创建用户额度记录异常",
				})
				return
			}
			tradeStatus = "TRADE_SUCCESS"
			// 添加额度变更记录
			model.RecordTopupLog(record.UserId, fmt.Sprintf("通过 stripe 充值 %s", common.LogQuota(int64(record.Quota))), 0)
		}
	}

	// 更新订单状态
	err = model.UpdateOrderStatusByTradeNo(sessionId, tradeStatus)
	if err != nil {
		logger.Error(c, "更新订单状态异常: "+err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	http.Redirect(c.Writer, c.Request, os.Getenv("WEB_DOMAIN")+"/topup?success=true", http.StatusSeeOther)
}

// Stripe订单支付失败
func StripeOrderFailed(c *gin.Context) {
	sessionId := c.Query("session_id")
	res, err := session.Get(sessionId, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if res.PaymentStatus == "unpaid" {
		err = model.UpdateOrderStatusByTradeNo(sessionId, "TRADE_CLOSED")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	http.Redirect(c.Writer, c.Request, os.Getenv("WEB_DOMAIN")+"/topup?canceled=true", http.StatusSeeOther)
}

// 查询Stripe订单状态
func QueryStripeOrder(c *gin.Context) {
	sessionId := c.Query("sessionId")
	res, err := session.Get(sessionId, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 获取订单信息
	record, err := model.GetOrderByTradeNo(sessionId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	tradeStatus := record.Status

	// 订单未支付
	if res.PaymentStatus == "unpaid" {
		tradeStatus = "TRADE_CLOSED"
	}

	// 支付成功，更新订单状态
	if res.PaymentStatus == "paid" {
		// 订单状态为等待付款时更新用户额度
		if tradeStatus == "WAIT_BUYER_PAY" {
			// 更新用户额度
			err = model.UpdateUserQuota(record.UserId, record.Quota)
			if err != nil {
				logger.Error(c, "更新用户额度异常: "+err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "更新用户额度异常",
				})
				return
			}
			err = model.AddQuotaRecord(record.UserId, 2, record.TradeNo, record.Quota)
			if err != nil {
				logger.Error(c, "创建用户额度记录异常: "+err.Error())
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "创建用户额度记录异常",
				})
				return
			}
			tradeStatus = "TRADE_SUCCESS"
			// 添加额度变更记录
			model.RecordTopupLog(record.UserId, fmt.Sprintf("通过 stripe 充值 %s", common.LogQuota(int64(record.Quota))), 0)
		}
	}

	if res.PaymentStatus == "no_payment_required" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "订单不存在或已过期",
		})
		return
	}

	// 更新订单状态
	err = model.UpdateOrderStatusByTradeNo(sessionId, tradeStatus)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单状态更新成功",
	})
}
