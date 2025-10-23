package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// AIProvider AI服务提供商
type AIProvider string

const (
	ProviderOpenAI AIProvider = "openai"
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
}

// GroupChatConfig 群聊AI配置
type GroupChatConfig struct {
	GroupID      int64                  `json:"group_id"`
	GroupName    string                 `json:"group_name"`
	GroupTopic   string                 `json:"group_topic"`
	ChatHistory  []ChatMessage          `json:"chat_history"`
	UserProfile  *UserProfile           `json:"user_profile"`
	AIPersona    string                 `json:"ai_persona"`
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
type ChatMessage struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	IsBot     bool      `json:"is_bot"`
}

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
	claudeKey    string
	customAPIURL string

	// 缓存和限制
	responseCache map[string]string
	requestLimit  int

	// 模型配置
	defaultModel string
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
		temperature:   0.7,
		maxTokens:     1000,
		topP:          1.0,
	}

	// 从配置中加载API密钥
	if key, ok := config["openai_key"].(string); ok {
		service.openAIKey = key
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

// generateOpenAIResponse 调用OpenAI API
func (s *aiService) generateOpenAIResponse(ctx context.Context, prompt string, maxLength int) (string, error) {
	// TODO: 实现OpenAI API调用
	// 这里需要使用OpenAI官方SDK或HTTP客户端
	s.logger.Debug("Using OpenAI API (mock implementation)")
	return s.generateFallbackResponse(prompt), nil
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
