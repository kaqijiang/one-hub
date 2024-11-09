package generalproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"one-api/common"
	"one-api/types"
	"strings"

	"github.com/gin-gonic/gin"
)

func (p *GeneralProxyProvider) RelayRequest(c *gin.Context) (*types.GeneralProxyResponse, *types.OpenAIErrorWithStatusCode) {
	// 根据 channel 配置构建完整的请求 URL
	fullURL := p.GetFullRequestURL(c)
	if fullURL == "" {
		return nil, common.ErrorWrapperLocal(nil, "invalid_general_proxy_config", http.StatusInternalServerError)
	}
	// 获取模型
	modelName := c.Request.Header.Get("OMINI-API-Model")

	// 删除多余 header
	c.Request.Header.Del("OMINI-API-Model")
	c.Request.Header.Del("Authorization")

	// 获取并设置 GeneralProxyProvider 中的请求头
	headers := p.GetRequestHeaders()

	// 复制原始请求的 Body
	var body io.Reader
	if c.Request.Body != nil {
		// 读取原始请求的 Body
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			return nil, common.ErrorWrapperLocal(err, "read_request_body_error", http.StatusInternalServerError)
		}
		// 创建一个新的 ReadCloser，以便后续处理
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		// 如果 modelName 以 "capcha" 开头，则修改 JSON body
		if strings.HasPrefix(strings.ToLower(modelName), "capcha") {

			// 解析 body 为 JSON 对象
			var jsonBody map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonBody); err != nil {
				return nil, common.ErrorWrapperLocal(err, "invalid_json_format", http.StatusBadRequest)
			}

			// 获取并删除指定的 appId 和 clientKey
			appId := headers["appId"]
			if _, exists := headers["appId"]; exists {
				delete(headers, "appId")
			}

			clientKey := headers["clientKey"]
			if _, exists := headers["clientKey"]; exists {
				delete(headers, "clientKey")
			}

			// 增加新的字段
			jsonBody["appId"] = appId
			jsonBody["clientKey"] = clientKey

			// 确保 "task" 是一个 map[string]interface{}
			task, ok := jsonBody["task"].(map[string]interface{})
			if ok {
				// 类型转换，确保 "type" 字段在 "task" 内部
				switch modelName {
				case "CapchaTurnstileTask":
					task["type"] = "AntiTurnstileTaskProxyLess"
				case "CapchaHCaptchaEnterpriseTask":
					task["type"] = "HCaptchaEnterpriseTaskProxyLess"
				case "CapchaReCaptchaV2Task":
					task["type"] = "ReCaptchaV2TaskProxyLess"
				case "CapchaReCaptchaV3Task":
					task["type"] = "ReCaptchaV3TaskProxyLess"
				case "CapchaReCaptchaV2EnterpriseTask":
					task["type"] = "ReCaptchaV2EnterpriseTaskProxyLess"
				case "CapchaReCaptchaV3EnterpriseTask":
					task["type"] = "ReCaptchaV3EnterpriseTaskProxyLess"
				case "CapchaGeeTestTask":
					task["type"] = "GeeTestTaskProxyLess"
				case "MTCaptchaTask":
					task["type"] = "MtCaptchaTaskProxyLess"
				case "AwsWafTask":
					task["type"] = "AntiAwsWafTaskProxyLess"
				default:
					return nil, common.ErrorWrapperLocal(err, "OMINI-API-Model is missing or does not match.", http.StatusBadRequest)
				}
			}

			// 重新将 JSON 对象编码回字节
			modifiedBodyBytes, err := json.Marshal(jsonBody)
			if err != nil {
				return nil, common.ErrorWrapperLocal(err, "json_encoding_error", http.StatusInternalServerError)
			}

			// 设置修改后的 body
			body = bytes.NewReader(modifiedBodyBytes)
		} else {
			// 不做修改，使用原始的 body
			body = bytes.NewReader(bodyBytes)
		}
	}

	// 创建新的请求
	req, err := http.NewRequest(c.Request.Method, fullURL, body)
	if err != nil {
		return nil, common.ErrorWrapperLocal(err, "request_creation_error", http.StatusInternalServerError)
	}

	// 设置请求头
	req.Header = c.Request.Header.Clone()

	// 如果存在 Auth-proxy 头部
	if authProxy := req.Header.Get("Auth-proxy"); authProxy != "" {
		// 删除 Auth-proxy 头部
		req.Header.Del("Auth-proxy")
		// 将 Auth-proxy 的值设置为新的 Authorization 头部
		req.Header.Set("Authorization", authProxy)
	}

	for k, v := range headers {
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
