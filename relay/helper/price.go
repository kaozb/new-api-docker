package helper

import (
	"fmt"
	"one-api/common"
	constant2 "one-api/constant"
	relaycommon "one-api/relay/common"
	"one-api/setting"
	"one-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

type GroupRatioInfo struct {
	GroupRatio        float64
	GroupSpecialRatio float64
}

type PriceData struct {
	ModelPrice             float64
	ModelRatio             float64
	CompletionRatio        float64
	CacheRatio             float64
	CacheCreationRatio     float64
	ImageRatio             float64
	UsePrice               bool
	ShouldPreConsumedQuota int
	GroupRatioInfo         GroupRatioInfo
}

func (p PriceData) ToSetting() string {
	return fmt.Sprintf("ModelPrice: %f, ModelRatio: %f, CompletionRatio: %f, CacheRatio: %f, GroupRatio: %f, UsePrice: %t, CacheCreationRatio: %f, ShouldPreConsumedQuota: %d, ImageRatio: %f", p.ModelPrice, p.ModelRatio, p.CompletionRatio, p.CacheRatio, p.GroupRatioInfo.GroupRatio, p.UsePrice, p.CacheCreationRatio, p.ShouldPreConsumedQuota, p.ImageRatio)
}

// HandleGroupRatio checks for "auto_group" in the context and updates the group ratio and relayInfo.Group if present
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) GroupRatioInfo {
	groupRatioInfo := GroupRatioInfo{
		GroupRatio:        1.0, // default ratio
		GroupSpecialRatio: 1.0, // default user group ratio
	}

	// check auto group
	autoGroup, exists := ctx.Get("auto_group")
	if exists {
		if common.DebugEnabled {
			println(fmt.Sprintf("final group: %s", autoGroup))
		}
		relayInfo.Group = autoGroup.(string)
	}

	// check user group special ratio
	userGroupRatio, ok := setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.Group)
	if ok {
		// user group special ratio
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
	} else {
		// normal group ratio
		groupRatioInfo.GroupRatio = setting.GetGroupRatio(relayInfo.Group)
	}

	return groupRatioInfo
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, maxTokens int) (PriceData, error) {
	modelPrice, usePrice := operation_setting.GetModelPrice(info.OriginModelName, false)

	groupRatioInfo := HandleGroupRatio(c, info)

	var preConsumedQuota int
	var modelRatio float64
	var completionRatio float64
	var cacheRatio float64
	var imageRatio float64
	var cacheCreationRatio float64
	if !usePrice {
		preConsumedTokens := common.PreConsumedQuota
		if maxTokens != 0 {
			preConsumedTokens = promptTokens + maxTokens
		}
		var success bool
		modelRatio, success = operation_setting.GetModelRatio(info.OriginModelName)
		if !success {
			acceptUnsetRatio := false
			if accept, ok := info.UserSetting[constant2.UserAcceptUnsetRatioModel]; ok {
				b, ok := accept.(bool)
				if ok {
					acceptUnsetRatio = b
				}
			}
			if !acceptUnsetRatio {
				return PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", info.OriginModelName, info.OriginModelName)
			}
		}
		completionRatio = operation_setting.GetCompletionRatio(info.OriginModelName)
		cacheRatio, _ = operation_setting.GetCacheRatio(info.OriginModelName)
		cacheCreationRatio, _ = operation_setting.GetCreateCacheRatio(info.OriginModelName)
		imageRatio, _ = operation_setting.GetImageRatio(info.OriginModelName)
		ratio := modelRatio * groupRatioInfo.GroupRatio
		preConsumedQuota = int(float64(preConsumedTokens) * ratio)
	} else {
		preConsumedQuota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	}

	priceData := PriceData{
		ModelPrice:             modelPrice,
		ModelRatio:             modelRatio,
		CompletionRatio:        completionRatio,
		GroupRatioInfo:         groupRatioInfo,
		UsePrice:               usePrice,
		CacheRatio:             cacheRatio,
		ImageRatio:             imageRatio,
		CacheCreationRatio:     cacheCreationRatio,
		ShouldPreConsumedQuota: preConsumedQuota,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}

	return priceData, nil
}

func ContainPriceOrRatio(modelName string) bool {
	_, ok := operation_setting.GetModelPrice(modelName, false)
	if ok {
		return true
	}
	_, ok = operation_setting.GetModelRatio(modelName)
	if ok {
		return true
	}
	return false
}
