package common

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"one-api/common"
	"strings"
)

func GetFullRequestURL(baseURL string, requestURL string, channelType int, info ...*RelayInfo) string {
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case common.ChannelTypeOpenAI:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case common.ChannelTypeAzure:
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}

	// 检查是否存在 maxkbConfig 配置
	if len(info) > 0 && info[0] != nil {
		if maxkbConfig, ok := info[0].ChannelSetting["maxkbConfig"]; ok {
			if configMap, ok := maxkbConfig.(map[string]interface{}); ok {
				// 根据 info.UpstreamModelName 查找对应的配置
				if modelConfig, ok := configMap[info[0].UpstreamModelName]; ok {
					if appKeyMap, ok := modelConfig.(map[string]interface{}); ok {
						// 取第一个 app:key 对
						for app, _ := range appKeyMap {
							fullRequestURL = fmt.Sprintf("%s/api/application/%s/chat/completions", baseURL, app)
							break
						}
					}
				}
			}
		}
	}

	return fullRequestURL
}

func GetAPIVersion(c *gin.Context) string {
	query := c.Request.URL.Query()
	apiVersion := query.Get("api-version")
	if apiVersion == "" {
		apiVersion = c.GetString("api_version")
	}
	return apiVersion
}
