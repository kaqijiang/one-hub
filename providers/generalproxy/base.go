package generalproxy

import (
	"fmt"
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

// 获取请求头
func (p *GeneralProxyProvider) GetRequestHeaders() map[string]string {
	headers := make(map[string]string)
	// 仅在 Key 存在时设置 Authorization
	if p.Channel.Key != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", p.Channel.Key)
	}
	return headers
}

// 获取完整的请求 URL
func (p *GeneralProxyProvider) GetFullRequestURL(requestURL string, _ string) string {
	baseURL := strings.TrimSuffix(p.Channel.GetBaseURL(), "/")
	// 剔除 /generalProxy
	proxiedPath := strings.TrimPrefix(requestURL, "/generalProxy")
	if !strings.HasPrefix(proxiedPath, "/") {
		proxiedPath = "/" + proxiedPath
	}
	return baseURL + proxiedPath
}
