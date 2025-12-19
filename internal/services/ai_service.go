package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
)

// AIProvider AI服务提供商
type AIProvider string

const (
	ProviderOpenAI AIProvider = "openai"
	ProviderGemini AIProvider = "gemini"
	ProviderClaude AIProvider = "claude"
	ProviderLocal  AIProvider = "local"
	ProviderCustom AIProvider = "custom"
)

// AIService AI服务接口
type AIService interface {
	GenerateGroupChatResponse(ctx context.Context, config *GroupChatConfig) (string, error)
	GeneratePrivateMessage(ctx context.Context, config *PrivateMessageConfig) (string, error)
	AnalyzeSentiment(ctx context.Context, text string) (*SentimentAnalysis, error)
	ExtractKeywords(ctx context.Context, text string) ([]string, error)
	GenerateVariations(ctx context.Context, template string, count int) ([]string, error)
	AgentDecision(ctx context.Context, req *models.AgentDecisionRequest) (*models.AgentDecisionResponse, error)
	GenerateImage(ctx context.Context, prompt string) (string, error)
}

// GroupChatConfig 群聊AI配置
type GroupChatConfig struct {
	GroupID     int64                `json:"group_id"`
	GroupName   string               `json:"group_name"`
	GroupTopic  string               `json:"group_topic"`
	ChatHistory []models.ChatMessage `json:"chat_history"`
	UserProfile *UserProfile         `json:"user_profile"`
	AIPersona   string               `json:"ai_persona"`

	ResponseType string                 `json:"response_type"` // casual, professional, humorous
	MaxLength    int                    `json:"max_length"`
	Language     string                 `json:"language"`
	Context      map[string]interface{} `json:"context"`
}

// PrivateMessageConfig 私信AI配置
type PrivateMessageConfig struct {
	TargetUser  *UserProfile           `json:"target_user"`
	MessageGoal string                 `json:"message_goal"` // sales, greeting, follow_up
	Industry    string                 `json:"industry"`
	Tone        string                 `json:"tone"` // friendly, professional, urgent
	Template    string                 `json:"template"`
	Variables   map[string]string      `json:"variables"`
	MaxLength   int                    `json:"max_length"`
	Language    string                 `json:"language"`
	Context     map[string]interface{} `json:"context"`
}

// ChatMessage 聊天消息
// type ChatMessage struct {
// 	UserID    int64     `json:"user_id"`
// 	Username  string    `json:"username"`
// 	Message   string    `json:"message"`
// 	Timestamp time.Time `json:"timestamp"`
// 	IsBot     bool      `json:"is_bot"`
// }

// UserProfile 用户档案
type UserProfile struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Bio       string `json:"bio"`
	Language  string `json:"language"`
	IsBot     bool   `json:"is_bot"`
	IsPremium bool   `json:"is_premium"`
}

// SentimentAnalysis 情感分析结果
type SentimentAnalysis struct {
	Sentiment  string   `json:"sentiment"`  // positive, negative, neutral
	Confidence float64  `json:"confidence"` // 0.0-1.0
	Emotions   []string `json:"emotions"`   // happy, sad, angry, etc.
	Keywords   []string `json:"keywords"`
	Toxicity   float64  `json:"toxicity"` // 0.0-1.0
	Intent     string   `json:"intent"`   // question, complaint, praise, etc.
}

// aiService AI服务实现
type aiService struct {
	provider AIProvider
	logger   *zap.Logger

	// AI服务配置
	openAIKey    string
	geminiKey    string
	claudeKey    string
	customAPIURL string

	// 缓存和限制
	responseCache map[string]string
	requestLimit  int

	// 模型配置
	defaultModel string
	geminiModel  string
	temperature  float64
	maxTokens    int
	topP         float64
}

// NewAIService 创建AI服务
func NewAIService(provider AIProvider, config map[string]interface{}) AIService {
	service := &aiService{
		provider:      provider,
		logger:        logger.Get().Named("ai_service"),
		responseCache: make(map[string]string),
		requestLimit:  100, // 每分钟100次请求
		defaultModel:  "gpt-3.5-turbo",
		geminiModel:   "gemini-2.5-flash",
		temperature:   0.7,
		maxTokens:     1000,
		topP:          1.0,
	}

	// 从配置中加载API密钥
	if key, ok := config["openai_key"].(string); ok {
		service.openAIKey = key
	}
	if key, ok := config["gemini_key"].(string); ok {
		service.geminiKey = key
	}
	if model, ok := config["gemini_model"].(string); ok && model != "" {
		service.geminiModel = model
	}
	if key, ok := config["claude_key"].(string); ok {
		service.claudeKey = key
	}
	if url, ok := config["custom_api_url"].(string); ok {
		service.customAPIURL = url
	}

	return service
}

// GenerateGroupChatResponse 生成群聊回复
func (s *aiService) GenerateGroupChatResponse(ctx context.Context, config *GroupChatConfig) (string, error) {
	s.logger.Info("Generating group chat response",
		zap.Int64("group_id", config.GroupID),
		zap.String("group_name", config.GroupName),
		zap.String("ai_persona", config.AIPersona))

	// 构建上下文
	contextPrompt := s.buildGroupChatContext(config)

	// 生成回复
	response, err := s.generateResponse(ctx, contextPrompt, config.MaxLength)
	if err != nil {
		s.logger.Error("Failed to generate group chat response", zap.Error(err))
		return "", err
	}

	// 后处理：确保回复符合群聊场景
	processedResponse := s.postProcessGroupChatResponse(response, config)

	s.logger.Info("Group chat response generated successfully",
		zap.Int("response_length", len(processedResponse)))

	return processedResponse, nil
}

// GeneratePrivateMessage 生成私信消息
func (s *aiService) GeneratePrivateMessage(ctx context.Context, config *PrivateMessageConfig) (string, error) {
	s.logger.Info("Generating private message",
		zap.String("message_goal", config.MessageGoal),
		zap.String("industry", config.Industry),
		zap.String("tone", config.Tone))

	// 构建消息上下文
	contextPrompt := s.buildPrivateMessageContext(config)

	// 生成消息
	response, err := s.generateResponse(ctx, contextPrompt, config.MaxLength)
	if err != nil {
		s.logger.Error("Failed to generate private message", zap.Error(err))
		return "", err
	}

	// 变量替换
	processedMessage := s.replaceVariables(response, config.Variables)

	s.logger.Info("Private message generated successfully",
		zap.Int("message_length", len(processedMessage)))

	return processedMessage, nil
}

// AnalyzeSentiment 情感分析
func (s *aiService) AnalyzeSentiment(ctx context.Context, text string) (*SentimentAnalysis, error) {
	s.logger.Debug("Analyzing sentiment", zap.String("text_preview", text[:min(len(text), 100)]))

	// 简单的情感分析实现（实际应该调用AI服务）
	analysis := &SentimentAnalysis{
		Sentiment:  s.detectSentiment(text),
		Confidence: 0.85,
		Emotions:   s.detectEmotions(text),
		Keywords:   s.extractSimpleKeywords(text),
		Toxicity:   s.detectToxicity(text),
		Intent:     s.detectIntent(text),
	}

	return analysis, nil
}

// ExtractKeywords 提取关键词
func (s *aiService) ExtractKeywords(ctx context.Context, text string) ([]string, error) {
	s.logger.Debug("Extracting keywords", zap.String("text_preview", text[:min(len(text), 100)]))

	// 简单的关键词提取（实际应该使用NLP库或AI服务）
	keywords := s.extractSimpleKeywords(text)
	return keywords, nil
}

// GenerateVariations 生成变体消息
func (s *aiService) GenerateVariations(ctx context.Context, template string, count int) ([]string, error) {
	s.logger.Info("Generating message variations",
		zap.String("template_preview", template[:min(len(template), 50)]),
		zap.Int("count", count))

	variations := make([]string, 0, count)

	for i := 0; i < count; i++ {
		prompt := fmt.Sprintf("请基于以下模板生成一个不同的表达方式，保持相同的意思但使用不同的词汇和句式：\n%s", template)

		variation, err := s.generateResponse(ctx, prompt, len(template)*2)
		if err != nil {
			s.logger.Error("Failed to generate variation", zap.Int("index", i), zap.Error(err))
			continue
		}

		variations = append(variations, variation)
	}

	return variations, nil
}

// AgentDecision 智能体决策
func (s *aiService) AgentDecision(ctx context.Context, req *models.AgentDecisionRequest) (*models.AgentDecisionResponse, error) {
	s.logger.Info("Generating agent decision",
		zap.String("persona", req.AgentPersona),
		zap.String("goal", req.AgentGoal))

	// 构建Prompt
	prompt := s.buildAgentDecisionPrompt(req)

	// 调用AI生成决策
	responseJSON, err := s.generateResponse(ctx, prompt, 1000)
	if err != nil {
		return nil, err
	}

	// 解析JSON响应
	// 这里假设AI返回的是合法的JSON字符串
	// 实际生产中可能需要更鲁棒的解析逻辑，处理Markdown代码块等
	cleanJSON := strings.TrimSpace(responseJSON)
	if strings.HasPrefix(cleanJSON, "```json") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	} else if strings.HasPrefix(cleanJSON, "```") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```")
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	var decision models.AgentDecisionResponse
	if err := json.Unmarshal([]byte(cleanJSON), &decision); err != nil {
		s.logger.Error("Failed to parse agent decision",
			zap.String("response", responseJSON),
			zap.Error(err))
		// 降级处理：如果不说话
		return &models.AgentDecisionResponse{ShouldSpeak: false}, nil
	}

	return &decision, nil
}

// OpenAI Image Generation Request
type openAIImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type openAIImageResponse struct {
	Data []struct {
		Url string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateImage 生成图片
func (s *aiService) GenerateImage(ctx context.Context, prompt string) (string, error) {
	s.logger.Info("Generating image", zap.String("prompt", prompt))

	if s.openAIKey == "" {
		s.logger.Warn("OpenAI key is missing, using placeholder image")
		return "https://via.placeholder.com/1024x1024.png?text=AI+Generated+Image", nil
	}

	reqBody := openAIImageRequest{
		Model:  "dall-e-3",
		Prompt: prompt,
		N:      1,
		Size:   "1024x1024",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/images/generations", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.openAIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result openAIImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("openai image api error: %s", result.Error.Message)
	}

	if len(result.Data) > 0 {
		return result.Data[0].Url, nil
	}

	return "", fmt.Errorf("no image generated")
}

// buildAgentDecisionPrompt 构建智能体决策Prompt
func (s *aiService) buildAgentDecisionPrompt(req *models.AgentDecisionRequest) string {
	var sb strings.Builder

	sb.WriteString("你是一个Telegram群组中的用户。请根据以下信息决定你的下一步行动。\n\n")

	sb.WriteString(fmt.Sprintf("当前话题: %s\n", req.ScenarioTopic))
	sb.WriteString(fmt.Sprintf("你的人设: %s\n", req.AgentPersona))
	sb.WriteString(fmt.Sprintf("你的目标: %s\n", req.AgentGoal))

	sb.WriteString("\n最近的聊天记录:\n")
	for _, msg := range req.ChatHistory {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp.Format("15:04"), msg.Username, msg.Message))
	}

	sb.WriteString("\n可用资源:\n")
	if len(req.ImagePool) > 0 {
		sb.WriteString(fmt.Sprintf("- 图片库: %d 张图片可用 (ID: 0-%d)\n", len(req.ImagePool), len(req.ImagePool)-1))
	} else {
		sb.WriteString("- 图片库: 无\n")
	}
	if req.ImageGenEnabled {
		sb.WriteString("- AI生图: 已启用 (可以使用 generate_photo 动作)\n")
	} else {
		sb.WriteString("- AI生图: 未启用\n")
	}

	sb.WriteString("\n请以JSON格式输出你的决策，包含以下字段:\n")
	sb.WriteString("- should_speak: boolean, 是否需要发言\n")
	sb.WriteString("- thought: string, 你的思考过程\n")
	sb.WriteString("- action: string, 动作类型 (send_text, send_photo, generate_photo)\n")
	sb.WriteString("- content: string, 发送的文本内容 (如果是发图，则是配文)\n")
	sb.WriteString("- media_path: string, 如果action是send_photo，填写图片库中的图片路径/索引\n")
	sb.WriteString("- image_prompt: string, 如果action是generate_photo，填写生图提示词\n")
	sb.WriteString("- reply_to_msg_id: int, 回复的消息ID (可选)\n")
	sb.WriteString("- delay_seconds: int, 模拟打字/思考延迟秒数 (1-10)\n")

	return sb.String()
}

// buildGroupChatContext 构建群聊上下文
func (s *aiService) buildGroupChatContext(config *GroupChatConfig) string {
	var contextBuilder strings.Builder

	// 基础上下文
	contextBuilder.WriteString(fmt.Sprintf("你现在是一个在Telegram群组'%s'中的用户。", config.GroupName))
	contextBuilder.WriteString(fmt.Sprintf("群组话题：%s\n", config.GroupTopic))
	contextBuilder.WriteString(fmt.Sprintf("你的人设：%s\n", config.AIPersona))
	contextBuilder.WriteString(fmt.Sprintf("回复风格：%s\n", config.ResponseType))

	// 聊天历史
	if len(config.ChatHistory) > 0 {
		contextBuilder.WriteString("\n最近的聊天记录：\n")
		for _, msg := range config.ChatHistory {
			contextBuilder.WriteString(fmt.Sprintf("[%s] %s: %s\n",
				msg.Timestamp.Format("15:04"), msg.Username, msg.Message))
		}
	}

	contextBuilder.WriteString(fmt.Sprintf("\n请根据以上上下文生成一个自然的群聊回复，长度不超过%d字符。回复应该：\n", config.MaxLength))
	contextBuilder.WriteString("1. 与聊天话题相关\n")
	contextBuilder.WriteString("2. 语气自然，符合人设\n")
	contextBuilder.WriteString("3. 避免重复之前的内容\n")
	contextBuilder.WriteString("4. 适当使用表情符号\n")

	return contextBuilder.String()
}

// buildPrivateMessageContext 构建私信上下文
func (s *aiService) buildPrivateMessageContext(config *PrivateMessageConfig) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(fmt.Sprintf("你需要写一条私信给%s。", config.TargetUser.FirstName))
	contextBuilder.WriteString(fmt.Sprintf("消息目标：%s\n", config.MessageGoal))
	contextBuilder.WriteString(fmt.Sprintf("行业背景：%s\n", config.Industry))
	contextBuilder.WriteString(fmt.Sprintf("语气风格：%s\n", config.Tone))

	if config.Template != "" {
		contextBuilder.WriteString(fmt.Sprintf("基于模板：%s\n", config.Template))
	}

	contextBuilder.WriteString(fmt.Sprintf("\n请生成一条长度不超过%d字符的私信，要求：\n", config.MaxLength))
	contextBuilder.WriteString("1. 语气友好自然\n")
	contextBuilder.WriteString("2. 内容简洁明了\n")
	contextBuilder.WriteString("3. 符合商务礼仪\n")
	contextBuilder.WriteString("4. 避免过于推销\n")

	return contextBuilder.String()
}

// generateResponse 生成AI回复的核心方法
func (s *aiService) generateResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	switch s.provider {
	case ProviderOpenAI:
		return s.generateOpenAIResponse(ctx, prompt, maxLength)
	case ProviderGemini:
		return s.generateGeminiResponse(ctx, prompt, maxLength)
	case ProviderClaude:
		return s.generateClaudeResponse(ctx, prompt, maxLength)
	case ProviderLocal:
		return s.generateLocalResponse(ctx, prompt, maxLength)
	case ProviderCustom:
		return s.generateCustomResponse(ctx, prompt, maxLength)
	default:
		return s.generateFallbackResponse(prompt), nil
	}
}

// OpenAI Chat Completion Request
type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// generateOpenAIResponse 调用OpenAI API
func (s *aiService) generateOpenAIResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	if s.openAIKey == "" {
		s.logger.Warn("OpenAI key is missing, using fallback response")
		return s.generateFallbackResponse(prompt), nil
	}

	reqBody := openAIChatRequest{
		Model: s.defaultModel,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: s.temperature,
		MaxTokens:   maxLength,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.openAIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("openai api error: %s", result.Error.Message)
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from openai")
}

// Gemini API Request/Response structures
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// generateGeminiResponse 调用Gemini API
func (s *aiService) generateGeminiResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	if s.geminiKey == "" {
		s.logger.Warn("Gemini key is missing, using fallback response")
		return s.generateFallbackResponse(prompt), nil
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     s.temperature,
			MaxOutputTokens: maxLength,
			TopP:            s.topP,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API URL (使用请求头认证方式)
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		s.geminiModel)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", s.geminiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("gemini api error: %s (code: %d)", result.Error.Message, result.Error.Code)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("no response from gemini")
}

// generateClaudeResponse 调用Claude API
func (s *aiService) generateClaudeResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	// TODO: 实现Claude API调用
	s.logger.Debug("Using Claude API (mock implementation)")
	return s.generateFallbackResponse(prompt), nil
}

// generateLocalResponse 使用本地模型
func (s *aiService) generateLocalResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	// TODO: 实现本地AI模型调用
	s.logger.Debug("Using local AI model (mock implementation)")
	return s.generateFallbackResponse(prompt), nil
}

// generateCustomResponse 使用自定义API
func (s *aiService) generateCustomResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	// TODO: 实现自定义API调用
	s.logger.Debug("Using custom API (mock implementation)")
	return s.generateFallbackResponse(prompt), nil
}

// generateFallbackResponse 备用响应生成
func (s *aiService) generateFallbackResponse(prompt string) string {
	// 简单的规则基础回复生成
	fallbackResponses := []string{
		"这是一个不错的观点！",
		"我觉得这个话题很有意思。",
		"确实如此，值得深入思考。",
		"感谢分享这个信息。",
		"这让我想到了类似的经历。",
	}

	// 基于prompt长度选择回复
	index := len(prompt) % len(fallbackResponses)
	return fallbackResponses[index]
}

// 辅助函数
func (s *aiService) postProcessGroupChatResponse(response string, config *GroupChatConfig) string {
	// 后处理逻辑：添加表情、调整长度等
	processed := strings.TrimSpace(response)

	// 长度限制
	if len(processed) > config.MaxLength {
		processed = processed[:config.MaxLength-3] + "..."
	}

	return processed
}

func (s *aiService) replaceVariables(message string, variables map[string]string) string {
	result := message
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (s *aiService) detectSentiment(text string) string {
	// 简单的情感检测逻辑
	positiveWords := []string{"好", "棒", "不错", "喜欢", "开心", "满意"}
	negativeWords := []string{"差", "糟", "讨厌", "生气", "失望", "难过"}

	positive, negative := 0, 0
	lowerText := strings.ToLower(text)

	for _, word := range positiveWords {
		if strings.Contains(lowerText, word) {
			positive++
		}
	}
	for _, word := range negativeWords {
		if strings.Contains(lowerText, word) {
			negative++
		}
	}

	if positive > negative {
		return "positive"
	} else if negative > positive {
		return "negative"
	}
	return "neutral"
}

func (s *aiService) detectEmotions(text string) []string {
	emotions := []string{}
	lowerText := strings.ToLower(text)

	emotionMap := map[string][]string{
		"happy":     {"开心", "高兴", "快乐", "兴奋"},
		"sad":       {"难过", "伤心", "沮丧", "失落"},
		"angry":     {"生气", "愤怒", "恼火", "愤恨"},
		"surprised": {"惊讶", "意外", "震惊", "惊奇"},
	}

	for emotion, keywords := range emotionMap {
		for _, keyword := range keywords {
			if strings.Contains(lowerText, keyword) {
				emotions = append(emotions, emotion)
				break
			}
		}
	}

	return emotions
}

func (s *aiService) extractSimpleKeywords(text string) []string {
	// 简单的关键词提取
	words := strings.Fields(text)
	keywords := []string{}

	// 过滤掉常见词汇
	stopWords := []string{"的", "是", "在", "有", "和", "了", "我", "你", "他", "她", "它"}

	for _, word := range words {
		if len(word) > 1 && !contains(stopWords, word) {
			keywords = append(keywords, word)
		}
	}

	// 返回前10个关键词
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

func (s *aiService) detectToxicity(text string) float64 {
	// 简单的毒性检测
	toxicWords := []string{"傻", "笨", "死", "滚", "垃圾"}
	lowerText := strings.ToLower(text)

	toxicCount := 0
	for _, word := range toxicWords {
		if strings.Contains(lowerText, word) {
			toxicCount++
		}
	}

	// 返回0.0-1.0之间的毒性评分
	return float64(toxicCount) / 10.0
}

func (s *aiService) detectIntent(text string) string {
	lowerText := strings.ToLower(text)

	if strings.Contains(lowerText, "?") || strings.Contains(lowerText, "？") {
		return "question"
	}
	if strings.Contains(lowerText, "谢谢") || strings.Contains(lowerText, "感谢") {
		return "gratitude"
	}
	if strings.Contains(lowerText, "抱怨") || strings.Contains(lowerText, "投诉") {
		return "complaint"
	}

	return "statement"
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
