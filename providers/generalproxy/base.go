package generalproxy

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"one-api/common/requester"
	"one-api/model"
	"one-api/providers/base"
	"one-api/types"
	"strings"
)

type GeneralProxyProviderFactory struct{}

func (f GeneralProxyProviderFactory) Create(channel *model.Channel) base.ProviderInterface {
	return &GeneralProxyProvider{
		BaseProvider: base.BaseProvider{
			Config:    getConfig(),
			Channel:   channel,
			Requester: requester.NewHTTPRequester(*channel.Proxy, requestErrorHandle),
		},
	}
}

type GeneralProxyProvider struct {
	base.BaseProvider
}

func getConfig() base.ProviderConfig {
	return base.ProviderConfig{
		BaseURL: "", // 我们将使用渠道的 BaseURL
	}
}

// 请求错误处理
func requestErrorHandle(resp *http.Response) *types.OpenAIError {
	// 根据需要解析响应，生成统一的错误对象
	return &types.OpenAIError{
		Message: "请求失败",
		Type:    "general_proxy_error",
		Code:    resp.StatusCode,
	}
}

func (p *GeneralProxyProvider) GetRequestHeaders() map[string]string {
	headers := make(map[string]string)

	// 如果 Key 存在且是 JSON 字符串，解析它
	if p.Channel.Key != "" {
		var parsedHeaders map[string]string
		// 解析 JSON，不再返回错误
		json.Unmarshal([]byte(p.Channel.Key), &parsedHeaders)

		// 将解析后的键值对添加到 headers 中
		for key, value := range parsedHeaders {
			headers[key] = value
		}
	}

	return headers
}

// 获取完整的请求 URL
func (p *GeneralProxyProvider) GetFullRequestURL(c *gin.Context) string {
	baseURL := strings.TrimSuffix(p.Channel.GetBaseURL(), "/")
	// 剔除 /generalProxy
	proxiedPath := strings.TrimPrefix(c.Request.URL.Path, "/generalProxy")
	if !strings.HasPrefix(proxiedPath, "/") {
		proxiedPath = "/" + proxiedPath
	}
	// 组合完整的 URL，包括查询参数
	fullURL := baseURL + proxiedPath
	if c.Request.URL.RawQuery != "" {
		fullURL += "?" + c.Request.URL.RawQuery
	}
	return fullURL
}
