package client

import (
	"crypto/tls"
	"github.com/songquanpeng/one-api/common/config"
	"net/http"
	"time"
)

var HTTPClient *http.Client
var ImpatientHTTPClient *http.Client

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if config.RelayTimeout == 0 {
		HTTPClient = &http.Client{
			Transport: tr,
		}
	} else {
		HTTPClient = &http.Client{
			Transport: tr,
			Timeout:   time.Duration(config.RelayTimeout) * time.Second,
		}
	}

	ImpatientHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
}
