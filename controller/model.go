package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel/ai360"
	"github.com/QuantumNous/new-api/relay/channel/lingyiwanwu"
	"github.com/QuantumNous/new-api/relay/channel/minimax"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// https://platform.openai.com/docs/api-reference/models/list

var openAIModels []dto.OpenAIModels
var openAIModelsMap map[string]dto.OpenAIModels
var channelId2Models map[int][]string

func init() {
	// https://platform.openai.com/docs/models/model-endpoint-compatibility
	for i := 0; i < constant.APITypeDummy; i++ {
		if i == constant.APITypeAIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName()
		modelNames := adaptor.GetModelList()
		for _, modelName := range modelNames {
			openAIModels = append(openAIModels, dto.OpenAIModels{
				Id:      modelName,
				Object:  "model",
				Created: 1626777600,
				OwnedBy: channelName,
			})
		}
	}
	for _, modelName := range ai360.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: ai360.ChannelName,
		})
	}
	for _, modelName := range moonshot.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: moonshot.ChannelName,
		})
	}
	for _, modelName := range lingyiwanwu.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: lingyiwanwu.ChannelName,
		})
	}
	for _, modelName := range minimax.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: minimax.ChannelName,
		})
	}
	openAIModelsMap = make(map[string]dto.OpenAIModels)
	for _, aiModel := range openAIModels {
		openAIModelsMap[aiModel.Id] = aiModel
	}
	channelId2Models = make(map[int][]string)
	for i := 1; i <= constant.ChannelTypeDummy; i++ {
		apiType, success := common.ChannelType2APIType(i)
		if !success || apiType == constant.APITypeAIProxyLibrary {
			continue
		}
		meta := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: i,
		}}
		adaptor := relay.GetAdaptor(apiType)
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
	}
	openAIModels = lo.UniqBy(openAIModels, func(m dto.OpenAIModels) string {
		return m.Id
	})
}

func channelOwnerName(channelType int) string {
	apiType, success := common.ChannelType2APIType(channelType)
	if !success {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType: channelType,
	}})
	if name := strings.TrimSpace(adaptor.GetChannelName()); name != "" {
		return name
	}
	return strings.ToLower(constant.GetChannelTypeName(channelType))
}

func getPreferredModelOwners(modelNames []string, groups []string) map[string]string {
	channelTypes, err := model.GetPreferredModelOwnerChannelTypes(modelNames, groups)
	if err != nil {
		common.SysLog(fmt.Sprintf("GetPreferredModelOwnerChannelTypes error: %v", err))
		return map[string]string{}
	}

	ownerByChannelType := make(map[int]string)
	owners := make(map[string]string, len(channelTypes))
	for modelName, channelType := range channelTypes {
		owner, ok := ownerByChannelType[channelType]
		if !ok {
			owner = channelOwnerName(channelType)
			ownerByChannelType[channelType] = owner
		}
		if owner != "" {
			owners[modelName] = owner
		}
	}
	return owners
}

func buildOpenAIModel(modelName string, ownerByModel map[string]string) dto.OpenAIModels {
	var oaiModel dto.OpenAIModels
	if staticModel, ok := openAIModelsMap[modelName]; ok {
		oaiModel = staticModel
	} else {
		oaiModel = dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: "custom",
		}
	}
	if owner, ok := ownerByModel[modelName]; ok && owner != "" {
		oaiModel.OwnedBy = owner
	}
	oaiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(modelName)
	return oaiModel
}

type modelListGroups struct {
	ownerGroups []string
}

type enabledModelChannelInfo struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Type         int    `json:"type"`
	TypeName     string `json:"type_name"`
	TestStatus   int    `json:"test_status"`
	TestTime     int64  `json:"test_time"`
	ResponseTime int    `json:"response_time"`
	TestError    string `json:"test_error"`
	TestResponse string `json:"test_response"`
}

type enabledModelMappingInfo struct {
	ChannelId    int    `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	TestStatus   int    `json:"test_status"`
	TestTime     int64  `json:"test_time"`
	ResponseTime int    `json:"response_time"`
	TestError    string `json:"test_error"`
	TestResponse string `json:"test_response"`
}

type enabledModelDetail struct {
	ModelName string                    `json:"model_name"`
	OwnedBy   string                    `json:"owned_by"`
	Channels  []enabledModelChannelInfo `json:"channels"`
	Mapped    bool                      `json:"mapped"`
	Mappings  []enabledModelMappingInfo `json:"mappings"`
}

func getModelListGroups(c *gin.Context) (modelListGroups, error) {
	return modelListGroups{
		ownerGroups: nil,
	}, nil
}

func ListModels(c *gin.Context, modelType int) {
	userModelNames := make([]string, 0)
	groups, err := getModelListGroups(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "get user group failed",
		})
		return
	}
	ownerGroups := groups.ownerGroups
	modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	if modelLimitEnable {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		var tokenModelLimit map[string]bool
		if ok {
			tokenModelLimit = s.(map[string]bool)
		} else {
			tokenModelLimit = map[string]bool{}
		}
		for allowModel, _ := range tokenModelLimit {
			userModelNames = append(userModelNames, allowModel)
		}
	} else {
		models := model.GetEnabledModels()
		for _, modelName := range models {
			userModelNames = append(userModelNames, modelName)
		}
	}

	ownerByModel := map[string]string{}
	ownerByModel = getPreferredModelOwners(userModelNames, ownerGroups)
	userOpenAiModels := make([]dto.OpenAIModels, 0, len(userModelNames))
	for _, modelName := range userModelNames {
		userOpenAiModels = append(userOpenAiModels, buildOpenAIModel(modelName, ownerByModel))
	}

	switch modelType {
	case constant.ChannelTypeAnthropic:
		useranthropicModels := make([]dto.AnthropicModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			useranthropicModels[i] = dto.AnthropicModel{
				ID:          model.Id,
				CreatedAt:   time.Unix(int64(model.Created), 0).UTC().Format(time.RFC3339),
				DisplayName: model.Id,
				Type:        "model",
			}
		}
		c.JSON(200, gin.H{
			"data":     useranthropicModels,
			"first_id": useranthropicModels[0].ID,
			"has_more": false,
			"last_id":  useranthropicModels[len(useranthropicModels)-1].ID,
		})
	case constant.ChannelTypeGemini:
		userGeminiModels := make([]dto.GeminiModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			userGeminiModels[i] = dto.GeminiModel{
				Name:        model.Id,
				DisplayName: model.Id,
			}
		}
		c.JSON(200, gin.H{
			"models":        userGeminiModels,
			"nextPageToken": nil,
		})
	default:
		c.JSON(200, gin.H{
			"success": true,
			"data":    userOpenAiModels,
			"object":  "list",
		})
	}
}

func ChannelListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    openAIModels,
	})
}

func DashboardListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    channelId2Models,
	})
}

func EnabledListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    model.GetEnabledModels(),
	})
}

func parseChannelModelMapping(modelMapping *string) map[string]string {
	if modelMapping == nil {
		return nil
	}
	rawMapping := strings.TrimSpace(*modelMapping)
	if rawMapping == "" || rawMapping == "{}" {
		return nil
	}
	parsed := make(map[string]string)
	if err := common.UnmarshalJsonStr(rawMapping, &parsed); err != nil {
		return nil
	}
	normalized := make(map[string]string, len(parsed))
	for source, target := range parsed {
		normalizedSource := strings.TrimSpace(source)
		normalizedTarget := strings.TrimSpace(target)
		if normalizedSource == "" || normalizedTarget == "" {
			continue
		}
		normalized[normalizedSource] = normalizedTarget
	}
	return normalized
}

func EnabledListModelDetails(c *gin.Context) {
	enabledChannels, err := model.GetEnabledModelChannels()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	detailMap := make(map[string]*enabledModelDetail)
	channelSetByModel := make(map[string]map[int]struct{})
	mappingSetByModel := make(map[string]map[string]struct{})

	for _, item := range enabledChannels {
		modelName := strings.TrimSpace(item.Model)
		if modelName == "" {
			continue
		}
		detail, ok := detailMap[modelName]
		if !ok {
			detail = &enabledModelDetail{
				ModelName: modelName,
				OwnedBy:   channelOwnerName(item.ChannelType),
				Channels:  []enabledModelChannelInfo{},
				Mappings:  []enabledModelMappingInfo{},
			}
			detailMap[modelName] = detail
			channelSetByModel[modelName] = make(map[int]struct{})
			mappingSetByModel[modelName] = make(map[string]struct{})
		}

		if _, exists := channelSetByModel[modelName][item.ChannelId]; !exists {
			channelSetByModel[modelName][item.ChannelId] = struct{}{}
			detail.Channels = append(detail.Channels, enabledModelChannelInfo{
				Id:           item.ChannelId,
				Name:         item.ChannelName,
				Type:         item.ChannelType,
				TypeName:     constant.GetChannelTypeName(item.ChannelType),
				TestStatus:   item.TestStatus,
				TestTime:     item.TestTime,
				ResponseTime: item.ResponseTime,
				TestError:    item.TestError,
				TestResponse: item.TestResponse,
			})
		}

		modelMapping := parseChannelModelMapping(item.ModelMapping)
		for source, target := range modelMapping {
			if source != modelName && target != modelName {
				continue
			}
			mappingKey := fmt.Sprintf("%d|%s|%s", item.ChannelId, source, target)
			if _, exists := mappingSetByModel[modelName][mappingKey]; exists {
				continue
			}
			mappingSetByModel[modelName][mappingKey] = struct{}{}
			detail.Mapped = true
			detail.Mappings = append(detail.Mappings, enabledModelMappingInfo{
				ChannelId:    item.ChannelId,
				ChannelName:  item.ChannelName,
				Source:       source,
				Target:       target,
				TestStatus:   item.TestStatus,
				TestTime:     item.TestTime,
				ResponseTime: item.ResponseTime,
				TestError:    item.TestError,
				TestResponse: item.TestResponse,
			})
		}
	}

	details := make([]enabledModelDetail, 0, len(detailMap))
	for _, detail := range detailMap {
		sort.Slice(detail.Channels, func(i, j int) bool {
			return detail.Channels[i].Id < detail.Channels[j].Id
		})
		sort.Slice(detail.Mappings, func(i, j int) bool {
			if detail.Mappings[i].ChannelId == detail.Mappings[j].ChannelId {
				return detail.Mappings[i].Source < detail.Mappings[j].Source
			}
			return detail.Mappings[i].ChannelId < detail.Mappings[j].ChannelId
		})
		details = append(details, *detail)
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].ModelName < details[j].ModelName
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    details,
	})
}

func RetrieveModel(c *gin.Context, modelType int) {
	modelId := c.Param("model")
	if aiModel, ok := openAIModelsMap[modelId]; ok {
		switch modelType {
		case constant.ChannelTypeAnthropic:
			c.JSON(200, dto.AnthropicModel{
				ID:          aiModel.Id,
				CreatedAt:   time.Unix(int64(aiModel.Created), 0).UTC().Format(time.RFC3339),
				DisplayName: aiModel.Id,
				Type:        "model",
			})
		default:
			c.JSON(200, aiModel)
		}
	} else {
		openAIError := types.OpenAIError{
			Message: fmt.Sprintf("The model '%s' does not exist", modelId),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
	}
}
