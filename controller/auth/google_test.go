package auth

import (
	"fmt"
	"testing"
)

// test google oauth
// @Summary test google oauth
// @Description test google oauth

func TestGetGoogleUserInfoByToken(t *testing.T) {
	user, err := GetGoogleUserInfoByToken("ya29.a0AXooCguC5tjJeVFLixgDwm4A9qhKMa7kIhq5HpTPbp97AfxQUHkCJdzXxnvjftPTGve2wBxT8ESljM3DDKap0DBbztivoGynHOHDvfkePjZSuu17k_IWeMhI6_712bimMj8OAhdg8wZm2qWOCqWeb0mJm4NeJ9xDbwIaCgYKAZ4SAQ8SFQHGX2MiI7mlfKoB8IRcCboXGK6TAg0170")

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(user.Id)
}
