package relay

import (
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/common/config"
	"one-api/common/logger"
	"one-api/model"
	providersBase "one-api/providers/base"
	"one-api/providers/generalproxy"
	"one-api/relay/relay_util"
	"one-api/types"

	"github.com/gin-gonic/gin"
)

// 处理代理请求并支持重试逻辑
func GeneralProxyRelay(c *gin.Context) {
	// 获取模型名称
	modelName := c.Request.Header.Get("OMINI-API-Model")
	if modelName == "" {
		common.AbortWithMessage(c, http.StatusBadRequest, "缺少模型名称")
		return
	}

	var lastError error

	retryTimes := config.RetryTimes

	// 进行请求和重试逻辑
	for attempt := 0; attempt <= retryTimes; attempt++ {
		// 获取提供者和模型名称
		provider, updatedModelName, err := GetProvider(c, modelName)
		if err != nil || provider == nil {
			lastError = err
			logger.LogError(c.Request.Context(), fmt.Sprintf("获取提供者失败，正在重试..."))
			continue
		}

		// 创建配额，基于每次请求消耗1次配额
		quota, errors := relay_util.NewQuota(c, modelName, 100)
		if errors != nil {
			common.AbortWithMessage(c, http.StatusForbidden, "请求额度已用尽、请充值令牌")
			return
		}

		modelName = updatedModelName

		channel := provider.GetChannel()
		// 冻结通道
		model.ChannelGroup.Cooldowns(channel.Id)

		// 处理单次代理请求
		errWithCode, done := processProxyRequest(c, provider, quota)
		if errWithCode == nil {
			return // 如果请求成功，直接返回
		}

		// 根据错误类型决定是否禁用通道
		if shouldDisableChannel(errWithCode) {
			channelId := c.GetInt("channel_id")
			channelName := "" // 从上下文或其他来源获取渠道名称
			processChannelRelayError(c.Request.Context(), channelId, channelName, errWithCode, config.ChannelTypeGeneralProxy)
		}
		// 如果请求失败，判断是否继续重试
		if done || !shouldRetryProxyRequest(c, errWithCode, channel.Type) {
			break // 请求完成或不再需要重试时退出重试逻辑
		}

		// 日志记录重试次数
		logger.LogError(c.Request.Context(), fmt.Sprintf("retrying with channel #%d (%s), remain retries: %d", channel.Id, channel.Name, retryTimes-attempt))
	}

	// 如果所有尝试均失败，返回最后一次错误
	common.AbortWithMessage(c, http.StatusBadGateway, lastError.Error())
}

// 处理单次代理请求
func processProxyRequest(c *gin.Context, provider providersBase.ProviderInterface, quota *relay_util.Quota) (*types.OpenAIErrorWithStatusCode, bool) {
	// 转换提供者为 GeneralProxyProvider
	gpProvider, ok := provider.(*generalproxy.GeneralProxyProvider)
	if !ok {
		return &types.OpenAIErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			OpenAIError: types.OpenAIError{
				Message: "无法转换为 GeneralProxyProvider",
			},
		}, true
	}

	// 转发请求
	proxyResponse, errWithCode := gpProvider.RelayRequest(c)
	if errWithCode == nil {
		defer proxyResponse.Response.Body.Close()

		// 消耗配额
		usage := &types.Usage{
			PromptTokens:     500,
			CompletionTokens: 500,
		}

		quota.Consume(c, usage)

		// 将响应复制回客户端
		for k, v := range proxyResponse.Response.Header {
			c.Writer.Header()[k] = v
		}
		c.Writer.WriteHeader(proxyResponse.Response.StatusCode)
		io.Copy(c.Writer, proxyResponse.Response.Body)

		//// 如果请求的是获取结果，回退配额
		//if proxyResponse.Response.Request.URL.Host == "api.capsolver.com" && proxyResponse.Response.Request.URL.Path == "/getTaskResult" {
		//	quota.Undo(c)
		//}
		return nil, true
	}

	// 请求失败，回退配额
	quota.Undo(c)
	return errWithCode, false
}

// 根据错误决定是否继续重试
func shouldRetryProxyRequest(c *gin.Context, errWithCode *types.OpenAIErrorWithStatusCode, channelType int) bool {
	// 如果错误状态码为 500 或者请求次数过多 (429)，则尝试重试
	if errWithCode.StatusCode >= 500 || errWithCode.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return false
}

// 根据错误决定是否禁用通道
func shouldDisableChannel(errWithCode *types.OpenAIErrorWithStatusCode) bool {
	// 如果错误状态码为 500 或者是请求次数过多 (429)，则禁用通道
	if errWithCode.StatusCode >= 500 || errWithCode.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return false
}