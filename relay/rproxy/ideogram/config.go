package ideogram

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

func SetHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("Api-Key", channel.Key)
	return nil
}
func GetName(path string) string {
	return strings.Join([]string{path, strconv.Itoa(int(channeltype.IdeoGram))}, "-")
}
func init() {
	//url-channeltype
	logger.SysLogf("register ideogram channel type start %s", channeltype.IdeoGram)
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc: SetHeaderFunc,
	}
	registry.Register(GetName("/generate"), adaptorBuilder)
	registry.Register(GetName("/edit"), adaptorBuilder)
	registry.Register(GetName("/remix"), adaptorBuilder)
	registry.Register(GetName("/upscale"), adaptorBuilder)
	registry.Register(GetName("/describe"), adaptorBuilder)
	registry.Register(GetName("/reframe"), adaptorBuilder)
	logger.SysLogf("register ideogram channel type end %s", channeltype.IdeoGram)

}
