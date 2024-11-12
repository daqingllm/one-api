package controller

import (
	"log"
	"net/smtp"

	"github.com/gin-gonic/gin"
	emailClient "github.com/jordan-wright/email"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

func SendEmail(c *gin.Context) {
	// 检查用户是否绑定邮箱
	email, err := model.GetUserEmail(c.GetInt(ctxkey.Id))
	if err != nil || email == "" {
		c.JSON(200, gin.H{
			"error": "no email found",
		})
		return
	}

	log.Println("send email init")
	//创建email
	e := emailClient.NewEmail()
	//设置发送方的邮箱
	e.From = "AiHubMix <chenxueamour@gmail.com>"
	// 设置接收方的邮箱
	e.To = []string{email}
	//设置主题
	e.Subject = "自动发送邮件测试"
	//设置文件发送的内容
	e.HTML = []byte(`
    <h1><a href="http://www.topgoer.com/">go语言中文网站</a></h1>    
    `)
	//设置服务器相关的配置
	// qq
	// err = e.Send("smtp.qq.com:25", smtp.PlainAuth("", "450907240@qq.com", "jsjbwqixmgiwcaaf", "smtp.qq.com"))
	// gmail
	err = e.Send("smtp.gmail.com:587", smtp.PlainAuth("", "chenxueamour@gmail.com", "sqvf kugg qiux gbou", "smtp.gmail.com"))
	if err != nil {
		log.Println("send email error", err)
	}
	c.JSON(200, gin.H{
		"message": "send email success",
	})
}
