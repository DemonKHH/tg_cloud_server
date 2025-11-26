package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"net/http"
	"strings"

	gotd_telegram "github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"sync"
	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
)

// AIService AI服务接口 (本地定义以避免循环引用)
type AIService interface {
	AgentDecision(ctx context.Context, req *models.AgentDecisionRequest) (*models.AgentDecisionResponse, error)
	GenerateImage(ctx context.Context, prompt string) (string, error)
}

// AgentRunner 智能体集群运行器
type AgentRunner struct {
	task           *models.Task
	scenario       *models.AgentScenario
	aiService      AIService
	connectionPool *ConnectionPool
	logger         *zap.Logger
	rnd            *rand.Rand

	// 消息缓存: accountID -> []ChatMessage
	messageCache map[string][]models.ChatMessage
	cacheMu      sync.RWMutex
}

// NewAgentRunner 创建智能体运行器
func NewAgentRunner(task *models.Task, aiService AIService, pool *ConnectionPool) (*AgentRunner, error) {
	// 解析场景配置
	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task config: %w", err)
	}

	var scenario models.AgentScenario
	if err := json.Unmarshal(configBytes, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse agent scenario: %w", err)
	}

	return &AgentRunner{
		task:           task,
		scenario:       &scenario,
		aiService:      aiService,
		connectionPool: pool,
		logger:         logger.Get().Named("agent_runner"),
		rnd:            rand.New(rand.NewSource(time.Now().UnixNano())),
		messageCache:   make(map[string][]models.ChatMessage),
	}, nil
}

// Run 运行智能体场景
func (r *AgentRunner) Run(ctx context.Context) error {
	r.logger.Info("Starting agent swarm scenario",
		zap.String("scenario", r.scenario.Name),
		zap.String("topic", r.scenario.Topic),
		zap.Int("agent_count", len(r.scenario.Agents)))

	// 验证所有账号可用性并注册消息监听
	for _, agent := range r.scenario.Agents {
		accountIDStr := fmt.Sprintf("%d", agent.AccountID)
		if !r.connectionPool.IsAccountBusy(accountIDStr) {
			// 注册更新处理器
			r.connectionPool.SetUpdateHandler(accountIDStr, r.createUpdateHandler(accountIDStr))
		}
	}

	// 运行主循环
	duration := time.Duration(r.scenario.Duration) * time.Second
	if duration == 0 {
		duration = 10 * time.Minute // 默认10分钟
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	ticker := time.NewTicker(5 * time.Second) // 每5秒进行一次调度检查
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			r.logger.Info("Scenario duration reached")
			return nil
		case <-ticker.C:
			r.scheduleAgents(ctx)
		}
	}
}

// scheduleAgents 调度智能体
func (r *AgentRunner) scheduleAgents(ctx context.Context) {
	// 随机选择一个智能体进行决策
	// 为了避免过于频繁，每次只选一个
	agentIndex := r.rnd.Intn(len(r.scenario.Agents))
	agent := r.scenario.Agents[agentIndex]

	// 检查活跃度
	if r.rnd.Float64() > agent.ActiveRate {
		return // 该智能体此次不活跃
	}

	r.logger.Debug("Agent selected for decision",
		zap.Uint64("account_id", agent.AccountID),
		zap.String("persona", agent.Persona.Name))

	// 执行决策循环
	if err := r.executeAgentLoop(ctx, &agent); err != nil {
		r.logger.Error("Agent execution failed",
			zap.Uint64("account_id", agent.AccountID),
			zap.Error(err))
	}
}

// executeAgentLoop 执行单个智能体的ODA循环
func (r *AgentRunner) executeAgentLoop(ctx context.Context, agent *models.AgentConfig) error {
	accountIDStr := fmt.Sprintf("%d", agent.AccountID)

	// 1. Observe (观察)
	// 获取最近的聊天记录
	// 这里需要通过 ConnectionPool 获取客户端并调用 API
	// 为了简化，我们假设可以通过 helper 方法获取
	history, err := r.fetchChatHistory(ctx, accountIDStr)
	if err != nil {
		return fmt.Errorf("failed to fetch chat history: %w", err)
	}

	// 2. Decide (决策)
	decisionReq := &models.AgentDecisionRequest{
		ScenarioTopic: r.scenario.Topic,
		AgentPersona: fmt.Sprintf("%s (Age: %d, Job: %s, Style: %v, Beliefs: %v)",
			agent.Persona.Name, agent.Persona.Age, agent.Persona.Occupation, agent.Persona.Style, agent.Persona.Beliefs),
		AgentGoal:       agent.Goal,
		ChatHistory:     history,
		ImagePool:       agent.ImagePool,
		ImageGenEnabled: agent.ImageGenEnabled,
	}

	decision, err := r.aiService.AgentDecision(ctx, decisionReq)
	if err != nil {
		return fmt.Errorf("AI decision failed: %w", err)
	}

	if !decision.ShouldSpeak {
		r.logger.Debug("Agent decided to stay silent",
			zap.String("thought", decision.Thought))
		return nil
	}

	r.logger.Info("Agent decided to act",
		zap.String("persona", agent.Persona.Name),
		zap.String("action", decision.Action),
		zap.String("thought", decision.Thought))

	// 3. Act (行动)
	// 模拟延迟
	delay := time.Duration(decision.DelaySeconds) * time.Second
	if delay == 0 {
		delay = time.Duration(r.rnd.Intn(5)+2) * time.Second
	}

	// 模拟输入状态
	r.simulateTyping(ctx, accountIDStr, delay)

	// 执行具体动作
	switch decision.Action {
	case "send_text":
		return r.sendTextMessage(ctx, accountIDStr, decision.Content, decision.ReplyToMsgID)
	case "send_photo":
		return r.sendPhotoFromPool(ctx, accountIDStr, decision.MediaPath, decision.Content)
	case "generate_photo":
		return r.generateAndSendPhoto(ctx, accountIDStr, decision.ImagePrompt, decision.Content)
	default:
		return r.sendTextMessage(ctx, accountIDStr, decision.Content, decision.ReplyToMsgID)
	}
}

// fetchChatHistory 获取聊天记录
func (r *AgentRunner) fetchChatHistory(ctx context.Context, accountID string) ([]models.ChatMessage, error) {
	// 1. 尝试从缓存获取
	r.cacheMu.RLock()
	cached, exists := r.messageCache[accountID]
	r.cacheMu.RUnlock()

	if exists && len(cached) > 0 {
		// 返回最近的20条
		if len(cached) > 20 {
			return cached[len(cached)-20:], nil
		}
		return cached, nil
	}

	// 2. 如果缓存为空，从API获取
	var history []models.ChatMessage

	task := &GenericTask{
		Type: "fetch_history",
		ExecuteFunc: func(ctx context.Context, client *gotd_telegram.Client) error {
			api := client.API()
			peer, err := r.resolvePeer(ctx, api, r.scenario.Topic)
			if err != nil {
				return err
			}

			messages, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
				Peer:  peer,
				Limit: 50, // 获取更多历史记录以填充缓存
			})
			if err != nil {
				return err
			}

			var messagesList []tg.MessageClass
			var usersList []tg.UserClass

			switch m := messages.(type) {
			case *tg.MessagesMessages:
				messagesList = m.Messages
				usersList = m.Users
			case *tg.MessagesMessagesSlice:
				messagesList = m.Messages
				usersList = m.Users
			case *tg.MessagesChannelMessages:
				messagesList = m.Messages
				usersList = m.Users
			}

			// Create a map of users for quick lookup
			usersMap := make(map[int64]*tg.User)
			for _, user := range usersList {
				if u, ok := user.(*tg.User); ok {
					usersMap[u.ID] = u
				}
			}

			for _, msg := range messagesList {
				if m, ok := msg.(*tg.Message); ok {
					chatMsg := models.ChatMessage{
						Message:   m.Message,
						Timestamp: time.Unix(int64(m.Date), 0),
						IsBot:     false,
					}
					// Resolve user info
					if fromID, ok := m.FromID.(*tg.PeerUser); ok {
						chatMsg.UserID = int64(fromID.UserID)
						if u, exists := usersMap[fromID.UserID]; exists {
							if u.Username != "" {
								chatMsg.Username = u.Username
							} else {
								chatMsg.Username = strings.TrimSpace(fmt.Sprintf("%s %s", u.FirstName, u.LastName))
							}
							chatMsg.IsBot = u.Bot
						}
					}
					history = append(history, chatMsg)
				}
			}
			// Reverse history to be chronological
			for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
				history[i], history[j] = history[j], history[i]
			}
			return nil
		},
	}

	err := r.connectionPool.ExecuteTask(accountID, task)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	r.cacheMu.Lock()
	r.messageCache[accountID] = history
	r.cacheMu.Unlock()

	// 返回最近的20条
	if len(history) > 20 {
		return history[len(history)-20:], nil
	}
	return history, nil
}

// createUpdateHandler 创建更新处理器
func (r *AgentRunner) createUpdateHandler(accountID string) gotd_telegram.UpdateHandler {
	return gotd_telegram.UpdateHandlerFunc(func(ctx context.Context, u tg.UpdatesClass) error {
		switch updates := u.(type) {
		case *tg.Updates:
			for _, update := range updates.Updates {
				r.handleUpdate(ctx, accountID, update, updates.Users)
			}
		case *tg.UpdatesCombined:
			for _, update := range updates.Updates {
				r.handleUpdate(ctx, accountID, update, updates.Users)
			}
		case *tg.UpdateShort:
			r.handleUpdate(ctx, accountID, updates.Update, nil)
		case *tg.UpdateShortMessage:
			// 简化处理，暂不处理ShortMessage
		case *tg.UpdateShortChatMessage:
			// 简化处理，暂不处理ShortChatMessage
		}
		return nil
	})
}

// handleUpdate 处理单个更新
func (r *AgentRunner) handleUpdate(ctx context.Context, accountID string, update tg.UpdateClass, users []tg.UserClass) {
	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			r.processNewMessage(accountID, msg, users)
		}
	case *tg.UpdateNewChannelMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			r.processNewMessage(accountID, msg, users)
		}
	}
}

// processNewMessage 处理新消息并更新缓存
func (r *AgentRunner) processNewMessage(accountID string, msg *tg.Message, users []tg.UserClass) {
	// 简单的用户查找表
	usersMap := make(map[int64]*tg.User)
	for _, user := range users {
		if u, ok := user.(*tg.User); ok {
			usersMap[u.ID] = u
		}
	}

	chatMsg := models.ChatMessage{
		Message:   msg.Message,
		Timestamp: time.Unix(int64(msg.Date), 0),
		IsBot:     false,
	}

	if fromID, ok := msg.FromID.(*tg.PeerUser); ok {
		chatMsg.UserID = int64(fromID.UserID)
		if u, exists := usersMap[fromID.UserID]; exists {
			if u.Username != "" {
				chatMsg.Username = u.Username
			} else {
				chatMsg.Username = strings.TrimSpace(fmt.Sprintf("%s %s", u.FirstName, u.LastName))
			}
			chatMsg.IsBot = u.Bot
		}
	}

	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// 追加新消息
	r.messageCache[accountID] = append(r.messageCache[accountID], chatMsg)

	// 限制缓存大小 (例如保留最近100条)
	if len(r.messageCache[accountID]) > 100 {
		r.messageCache[accountID] = r.messageCache[accountID][len(r.messageCache[accountID])-100:]
	}

	r.logger.Debug("New message cached",
		zap.String("account_id", accountID),
		zap.String("sender", chatMsg.Username),
		zap.String("content", chatMsg.Message))
}

// simulateTyping 模拟输入状态
func (r *AgentRunner) simulateTyping(ctx context.Context, accountID string, duration time.Duration) {
	task := &GenericTask{
		Type: "simulate_typing",
		ExecuteFunc: func(ctx context.Context, client *gotd_telegram.Client) error {
			api := client.API()
			peer, err := r.resolvePeer(ctx, api, r.scenario.Topic)
			if err != nil {
				return err
			}
			_, err = api.MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
				Peer:   peer,
				Action: &tg.SendMessageTypingAction{},
			})
			return err
		},
	}
	r.connectionPool.ExecuteTask(accountID, task)
	time.Sleep(duration)
}

// sendTextMessage 发送文本消息
func (r *AgentRunner) sendTextMessage(ctx context.Context, accountID string, content string, replyTo int64) error {
	task := &GenericTask{
		Type: "send_text",
		ExecuteFunc: func(ctx context.Context, client *gotd_telegram.Client) error {
			api := client.API()
			peer, err := r.resolvePeer(ctx, api, r.scenario.Topic)
			if err != nil {
				return err
			}

			req := &tg.MessagesSendMessageRequest{
				Peer:     peer,
				Message:  content,
				RandomID: time.Now().UnixNano(),
			}
			if replyTo != 0 {
				req.ReplyTo = &tg.InputReplyToMessage{ReplyToMsgID: int(replyTo)}
			}

			_, err = api.MessagesSendMessage(ctx, req)
			return err
		},
	}
	return r.connectionPool.ExecuteTask(accountID, task)
}

// sendPhotoFromPool 发送图片池中的图片
func (r *AgentRunner) sendPhotoFromPool(ctx context.Context, accountID string, mediaPath string, caption string) error {
	task := &GenericTask{
		Type: "send_photo",
		ExecuteFunc: func(ctx context.Context, client *gotd_telegram.Client) error {
			api := client.API()
			sender := message.NewSender(api).WithUploader(uploader.NewUploader(api))

			peer, err := r.resolvePeer(ctx, api, r.scenario.Topic)
			if err != nil {
				return err
			}

			var file tg.InputFileClass
			if strings.HasPrefix(mediaPath, "http") {
				resp, err := http.Get(mediaPath)
				if err != nil {
					return fmt.Errorf("failed to download image: %w", err)
				}
				defer resp.Body.Close()

				u := uploader.NewUploader(api)
				upload, err := u.Upload(ctx, uploader.NewUpload(mediaPath, resp.Body, resp.ContentLength))
				if err != nil {
					return fmt.Errorf("failed to upload image: %w", err)
				}
				file = upload
			} else {
				u := uploader.NewUploader(api)
				upload, err := u.FromPath(ctx, mediaPath)
				if err != nil {
					return fmt.Errorf("failed to upload local image: %w", err)
				}
				file = upload
			}

			_, err = sender.To(peer).Media(ctx, message.UploadedPhoto(file, styling.Plain(caption)))
			return err
		},
	}
	return r.connectionPool.ExecuteTask(accountID, task)
}

// resolvePeer 解析目标Peer
func (r *AgentRunner) resolvePeer(ctx context.Context, api *tg.Client, target string) (tg.InputPeerClass, error) {
	// Simple username resolution
	cleanTarget := strings.TrimPrefix(target, "@")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://t.me/")
	cleanTarget = strings.TrimPrefix(cleanTarget, "t.me/")

	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: cleanTarget,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve peer %s: %w", target, err)
	}

	if len(resolved.Chats) > 0 {
		if chat, ok := resolved.Chats[0].(*tg.Chat); ok {
			return &tg.InputPeerChat{ChatID: chat.ID}, nil
		} else if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
			return &tg.InputPeerChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}, nil
		}
	}
	return nil, fmt.Errorf("peer not found: %s", target)
}

// GenericTask 通用任务封装
type GenericTask struct {
	ExecuteFunc func(ctx context.Context, client *gotd_telegram.Client) error
	Type        string
}

func (t *GenericTask) Execute(ctx context.Context, api *tg.Client) error {
	return fmt.Errorf("use ExecuteAdvanced")
}

func (t *GenericTask) ExecuteAdvanced(ctx context.Context, client *gotd_telegram.Client) error {
	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, client)
	}
	return nil
}

func (t *GenericTask) GetType() string {
	return t.Type
}

// generateAndSendPhoto 生成并发送图片
func (r *AgentRunner) generateAndSendPhoto(ctx context.Context, accountID string, prompt string, caption string) error {
	// 1. 调用 AI 生成图片
	imageURL, err := r.aiService.GenerateImage(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	r.logger.Info("Image generated", zap.String("url", imageURL))

	// 2. 发送生成的图片
	// 这里通常需要下载图片然后上传，或者直接发送 URL (如果支持)
	return r.sendPhotoFromPool(ctx, accountID, imageURL, caption)
}
