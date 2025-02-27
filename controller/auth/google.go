package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/model"
)

type GoogleUser struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

func GetGoogleUserInfoByToken(ctx context.Context, access_token string) (*GoogleUser, error) {
	if access_token == "" {
		return nil, errors.New("无效的参数")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo?access_token="+access_token, nil)
	if err != nil {
		logger.Error(ctx, "Failed to create request "+err.Error())
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		logger.Error(ctx, "Failed to request Google API "+err.Error())
		return nil, errors.New("无法连接至 Google 服务器，请稍后重试！")
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error(ctx, "Failed to close response body "+err.Error())
		}
	}(res.Body)
	var googleUser GoogleUser
	err = json.NewDecoder(res.Body).Decode(&googleUser)
	if err != nil {
		return nil, err
	}
	if googleUser.Id == "" {
		return nil, errors.New("返回值非法，用户字段为空，请稍后重试！")
	}
	return &googleUser, nil
}

func GoogleOAuth(c *gin.Context) {
	session := sessions.Default(c)
	state := c.Query("state")
	if state == "" || session.Get("oauth_state") == nil || state != session.Get("oauth_state").(string) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "state is empty or not same",
		})
		return
	}
	username := session.Get("username")
	if username != nil {
		GoogleBind(c)
		return
	}

	code := c.Query("code")
	affCode := c.Query("aff_code")
	googleUser, err := GetGoogleUserInfoByToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user := model.User{
		GoogleId: googleUser.Id,
	}
	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		err := user.FillUserByGoogleId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	} else {
		if config.RegisterEnabled {
			username, err := model.GetRandomUserName()
			if err != nil || username == "" {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": "无法生成随机用户名",
				})
				return
			}
			user.Username = username
			user.DisplayName = "Google User"
			inviterId, _ := model.GetUserIdByAffCode(affCode)
			user.GoogleId = googleUser.Id
			user.Email = googleUser.Email
			user.Role = model.RoleCommonUser
			user.Status = model.UserStatusEnabled
			user.InviterId = inviterId

			if err := user.Insert(inviterId); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != model.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}
	controller.SetupLogin(&user, c)
}

func GoogleBind(c *gin.Context) {
	code := c.Query("code")
	googleUser, err := GetGoogleUserInfoByToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user := model.User{
		GoogleId: googleUser.Id,
	}
	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Google 账户已被绑定",
		})
		return
	}
	session := sessions.Default(c)
	id := session.Get("id")
	// id := c.GetInt("id")  // critical bug!
	user.Id = id.(int)
	err = user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.GoogleId = googleUser.Id
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
	return
}
