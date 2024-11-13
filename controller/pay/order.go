package pay

import (
	"net/http"
	"strconv"

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
func UpdateAllOrderStatus(c *gin.Context) {
	orders, err := model.GetOrdersByStatus("WAIT_BUYER_PAY")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 遍历订单，查询支付宝订单状态并更新
	for _, order := range orders {
		if order.GrantType == 1 {
			// 查询支付宝订单状态
			_, err := QueryAlipayOrder(c, order.TradeNo)
			if err != nil {
				continue
			}
		} else if order.GrantType == 2 {
			// 查询Stripe订单状态
			// _, err := QueryStripeOrder(c, order.TradeNo)
			// if err != nil {
			continue
			// }
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": strconv.Itoa(len(orders)) + "条订单状态更新成功",
	})
}
