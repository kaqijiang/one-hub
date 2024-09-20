package generalproxy

import (
	"net/http"
	"one-api/common"
	"one-api/types"

	"github.com/gin-gonic/gin"
)

func (p *GeneralProxyProvider) RelayRequest(c *gin.Context) (*types.GeneralProxyResponse, *types.OpenAIErrorWithStatusCode) {
	// 根据 channel 配置构建完整的请求 URL
	fullURL := p.GetFullRequestURL(c.Request.URL.Path, "")
	if fullURL == "" {
		return nil, common.ErrorWrapperLocal(nil, "invalid_general_proxy_config", http.StatusInternalServerError)
	}

	// 创建新的请求
	req, err := http.NewRequest(c.Request.Method, fullURL, c.Request.Body)
	if err != nil {
		return nil, common.ErrorWrapperLocal(err, "request_creation_error", http.StatusInternalServerError)
	}
	defer req.Body.Close()

	// 设置请求头
	req.Header = c.Request.Header.Clone()
	// 移除 X-API-Model 请求头
	req.Header.Del("X-API-Model")
	for k, v := range p.GetRequestHeaders() {
		req.Header.Set(k, v)
	}

	// 发送请求
	resp, errWithCode := p.Requester.SendRequestRaw(req)
	if errWithCode == nil {
		// 请求成功
		usage := p.calculateUsage(c, resp)
		p.SetUsage(usage)

		proxyResponse := &types.GeneralProxyResponse{
			Response: resp,
		}
		return proxyResponse, nil
	}

	// 请求失败，返回错误
	return nil, errWithCode
}

func (p *GeneralProxyProvider) calculateUsage(c *gin.Context, resp *http.Response) *types.Usage {
	// 根据请求和响应的字节数计算
	requestSize := c.Request.ContentLength
	responseSize := resp.ContentLength

	// 假设每 1KB 计为 1 个 Token
	promptTokens := int(requestSize / 1024)
	completionTokens := int(responseSize / 1024)
	totalTokens := promptTokens + completionTokens

	return &types.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
	}
}
