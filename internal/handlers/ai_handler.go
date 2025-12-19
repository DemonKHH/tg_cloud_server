package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/services"
)

// AIHandler AI服务处理器
type AIHandler struct {
	aiService services.AIService
	logger    *zap.Logger
}

// NewAIHandler 创建AI处理器
func NewAIHandler(aiService services.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		logger:    logger.Get().Named("ai_handler"),
	}
}

// GenerateGroupChatResponse 生成群聊AI回复
// @Summary 生成群聊AI回复
// @Description 根据群聊历史和配置生成智能回复
// @Tags AI服务
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body services.GroupChatConfig true "群聊AI配置"
// @Success 200 {object} map[string]string "生成的回复内容"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/group-chat [post]
func (h *AIHandler) GenerateGroupChatResponse(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var config services.GroupChatConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	// 设置默认值
	if config.MaxLength == 0 {
		config.MaxLength = 200
	}
	if config.Language == "" {
		config.Language = "zh-CN"
	}
	if config.ResponseType == "" {
		config.ResponseType = "casual"
	}

	aiResponse, err := h.aiService.GenerateGroupChatResponse(c.Request.Context(), &config)
	if err != nil {
		h.logger.Error("Failed to generate group chat response", zap.Error(err))
		response.InternalError(c, "生成AI回复失败")
		return
	}

	response.Success(c, gin.H{
		"response": aiResponse,
		"metadata": gin.H{
			"length":        len(aiResponse),
			"response_type": config.ResponseType,
			"language":      config.Language,
		},
	})
}

// GeneratePrivateMessage 生成私信内容
// @Summary 生成私信内容
// @Description 根据目标和配置生成个性化私信内容
// @Tags AI服务
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body services.PrivateMessageConfig true "私信AI配置"
// @Success 200 {object} map[string]string "生成的私信内容"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/private-message [post]
func (h *AIHandler) GeneratePrivateMessage(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var config services.PrivateMessageConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	// 设置默认值
	if config.MaxLength == 0 {
		config.MaxLength = 300
	}
	if config.Language == "" {
		config.Language = "zh-CN"
	}
	if config.MessageGoal == "" {
		config.MessageGoal = "greeting"
	}

	message, err := h.aiService.GeneratePrivateMessage(c.Request.Context(), &config)
	if err != nil {
		h.logger.Error("Failed to generate private message", zap.Error(err))
		response.InternalError(c, "生成私信内容失败")
		return
	}

	response.Success(c, gin.H{
		"message": message,
		"metadata": gin.H{
			"length":       len(message),
			"message_goal": config.MessageGoal,
			"language":     config.Language,
		},
	})
}

// AnalyzeSentiment 分析文本情感
// @Summary 分析文本情感
// @Description 分析输入文本的情感倾向和置信度
// @Tags AI服务
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string]string true "文本分析请求" example:{"text":"这是一个很棒的产品！"}
// @Success 200 {object} services.SentimentAnalysis "情感分析结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/analyze-sentiment [post]
func (h *AIHandler) AnalyzeSentiment(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	if len(req.Text) > 1000 {
		response.InvalidParam(c, "文本长度超过1000个字符")
		return
	}

	analysis, err := h.aiService.AnalyzeSentiment(c.Request.Context(), req.Text)
	if err != nil {
		h.logger.Error("Failed to analyze sentiment", zap.Error(err))
		response.InternalError(c, "情感分析失败")
		return
	}

	response.Success(c, analysis)
}

// ExtractKeywords 提取关键词
// @Summary 提取文本关键词
// @Description 从输入文本中提取重要关键词
// @Tags AI服务
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string]string true "关键词提取请求" example:{"text":"人工智能技术在现代社会中发挥着重要作用"}
// @Success 200 {object} map[string][]string "提取的关键词列表"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/extract-keywords [post]
func (h *AIHandler) ExtractKeywords(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	if len(req.Text) > 2000 {
		response.InvalidParam(c, "文本长度超过2000个字符")
		return
	}

	keywords, err := h.aiService.ExtractKeywords(c.Request.Context(), req.Text)
	if err != nil {
		h.logger.Error("Failed to extract keywords", zap.Error(err))
		response.InternalError(c, "关键词提取失败")
		return
	}

	response.Success(c, gin.H{
		"keywords": keywords,
		"count":    len(keywords),
	})
}

// GenerateVariations 生成模板变体
// @Summary 生成模板变体
// @Description 基于输入模板生成多个不同的变体版本
// @Tags AI服务
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string]interface{} true "变体生成请求" example:{"template":"欢迎加入我们的社群！","count":3}
// @Success 200 {object} map[string][]string "生成的变体列表"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/generate-variations [post]
func (h *AIHandler) GenerateVariations(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		Template string `json:"template" binding:"required"`
		Count    int    `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	// 设置默认值和限制
	if req.Count == 0 {
		req.Count = 3
	}
	if req.Count > 10 {
		req.Count = 10 // 最多生成10个变体
	}

	if len(req.Template) > 500 {
		response.InvalidParam(c, "模板长度超过500个字符")
		return
	}

	variations, err := h.aiService.GenerateVariations(c.Request.Context(), req.Template, req.Count)
	if err != nil {
		h.logger.Error("Failed to generate variations", zap.Error(err))
		response.InternalError(c, "生成模板变体失败")
		return
	}

	response.Success(c, gin.H{
		"variations":      variations,
		"generated_count": len(variations),
		"requested_count": req.Count,
	})
}

// GetAIConfig 获取AI服务配置
// @Summary 获取AI服务配置
// @Description 获取当前AI服务的配置信息和可用功能
// @Tags AI服务
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "AI服务配置信息"
// @Failure 401 {object} map[string]string "未授权"
// @Router /api/v1/ai/config [get]
func (h *AIHandler) GetAIConfig(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	config := gin.H{
		"providers": []string{"openai", "claude", "local", "custom"},
		"features": gin.H{
			"group_chat":      true,
			"private_message": true,
			"sentiment":       true,
			"keywords":        true,
			"variations":      true,
		},
		"limits": gin.H{
			"max_text_length":     2000,
			"max_template_length": 500,
			"max_variations":      10,
			"max_chat_history":    50,
		},
		"supported_languages": []string{
			"zh-CN", "en-US", "ja-JP", "ko-KR", "es-ES", "fr-FR", "de-DE", "ru-RU",
		},
		"response_types": []string{
			"casual", "professional", "humorous", "formal", "friendly",
		},
		"message_goals": []string{
			"greeting", "sales", "follow_up", "support", "engagement",
		},
	}

	response.Success(c, config)
}

// TestAIService 测试AI服务连接
// @Summary 测试AI服务连接
// @Description 测试AI服务是否可用，包括AI生成能力测试
// @Tags AI服务
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "测试结果"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/ai/test [post]
func (h *AIHandler) TestAIService(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 执行简单的AI功能测试
	testText := "测试AI服务连接"

	// 测试情感分析
	sentimentResult, sentimentErr := h.aiService.AnalyzeSentiment(c.Request.Context(), testText)

	// 测试关键词提取
	keywordsResult, keywordsErr := h.aiService.ExtractKeywords(c.Request.Context(), testText)

	// 测试AI生成能力（调用实际的AI API）
	generateConfig := &services.GroupChatConfig{
		GroupName:    "测试群组",
		GroupTopic:   "技术讨论",
		AIPersona:    "友好的技术爱好者",
		ResponseType: "casual",
		MaxLength:    100,
		Language:     "zh",
	}
	generateResult, generateErr := h.aiService.GenerateGroupChatResponse(c.Request.Context(), generateConfig)

	result := gin.H{
		"service_status": "available",
		"tests": gin.H{
			"sentiment_analysis": gin.H{
				"success": sentimentErr == nil,
				"result":  sentimentResult,
				"error":   getErrorString(sentimentErr),
			},
			"keyword_extraction": gin.H{
				"success": keywordsErr == nil,
				"result":  keywordsResult,
				"error":   getErrorString(keywordsErr),
			},
			"ai_generation": gin.H{
				"success": generateErr == nil,
				"result":  generateResult,
				"error":   getErrorString(generateErr),
			},
		},
		"timestamp": c.GetHeader("X-Request-ID"),
	}

	// 如果AI生成测试失败，标记为部分可用
	if generateErr != nil {
		result["service_status"] = "partial"
		h.logger.Warn("AI generation test failed", zap.Error(generateErr))
	}

	// 如果所有测试都失败，返回错误状态
	if sentimentErr != nil && keywordsErr != nil && generateErr != nil {
		result["service_status"] = "unavailable"
		response.Error(c, response.CodeInternalError, "AI服务不可用")
		return
	}

	response.Success(c, result)
}

// getErrorString 获取错误字符串，如果错误为nil则返回nil
func getErrorString(err error) interface{} {
	if err != nil {
		return err.Error()
	}
	return nil
}
