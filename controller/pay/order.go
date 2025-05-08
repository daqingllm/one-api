package pay

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

type CreateOrderRequest struct {
	Amount      float64 `json:"amount"`
	TotalAmount float64 `json:"totalAmount"`
}

// 查询用户订单状态
func QueryOrderByTradeNo(c *gin.Context) {
	tradeNo := c.Query("tradeNo")
	record, err := model.GetOrderByTradeNo(tradeNo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if record.IsOrderExpired() {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "订单已过期",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单信息",
		"data":    record,
	})
}

// 查询订单表所有等待付款的订单 再次查询支付宝订单状态并更新
func UpdateOrderStatusByUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Query("userId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	// 没有传入hours参数，默认查询48小时内的订单
	hours := 48
	if c.Query("hours") != "" {
		hours, err = strconv.Atoi(c.Query("hours"))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	expiredAt := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
	orders, err := model.GetOrdersByStatusByUserId(userId, "WAIT_BUYER_PAY", expiredAt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	updatedOrders := make([]*model.OrderRecord, 0)
	errList := make([]string, 0)
	// 遍历订单，查询支付宝订单状态并更新
	for _, order := range orders {
		if order.GrantType == 1 {
			// 查询支付宝订单状态
			updated, err := QueryAlipayOrder(c, order.TradeNo)
			if err != nil {
				errList = append(errList, order.TradeNo+" 支付宝订单查询失败："+err.Error())
				continue
			}
			if updated {
				updatedOrders = append(updatedOrders, order)
			}
		} else if order.GrantType == 2 {
			// 查询Stripe订单状态
			updated, err := QueryStripeOrder(c, order.TradeNo)
			if err != nil {
				errList = append(errList, order.TradeNo+" Stripe订单查询失败："+err.Error())
				continue
			}
			if updated {
				updatedOrders = append(updatedOrders, order)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "查询到" + strconv.Itoa(len(orders)) + "条未完成订单，" + strconv.Itoa(len(updatedOrders)) + "条订单状态更新成功",
		"data":    updatedOrders,
		"error":   errList,
	})
}
