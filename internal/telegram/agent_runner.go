package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	gotd_telegram "github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
)

// AIService AI服务接口 (本地定义以避免循环引用)
type AIService interface {
	AgentDecision(ctx context.Context, req *models.AgentDecisionRequest) (*models.AgentDecisionResponse, error)
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
	startTime := time.Now()
	r.logger.Info("Starting agent swarm scenario",
		zap.String("scenario", r.scenario.Name),
		zap.String("topic", r.scenario.Topic),
		zap.Int("agent_count", len(r.scenario.Agents)),
		zap.Int("duration_seconds", r.scenario.Duration))

	// 首先让所有智能体加入目标群组
	if r.scenario.Topic != "" {
		r.logger.Info("Ensuring all agents join the target group", zap.String("topic", r.scenario.Topic))
		for _, agent := range r.scenario.Agents {
			accountIDStr := fmt.Sprintf("%d", agent.AccountID)
			if err := r.ensureJoinGroup(ctx, accountIDStr, r.scenario.Topic); err != nil {
				r.logger.Warn("Failed to join group for agent",
					zap.Uint64("account_id", agent.AccountID),
					zap.String("topic", r.scenario.Topic),
					zap.Error(err))
				// 继续尝试其他账号，不中断整个任务
			} else {
				r.logger.Info("Agent joined group successfully",
					zap.Uint64("account_id", agent.AccountID),
					zap.String("topic", r.scenario.Topic))
			}
			// 加入群组之间稍微等待，避免频率限制
			time.Sleep(2 * time.Second)
		}
	}

	// 注册消息监听（无论账号是否忙碌，场景任务需要监听消息）
	registeredCount := 0
	for _, agent := range r.scenario.Agents {
		accountIDStr := fmt.Sprintf("%d", agent.AccountID)
		// 注册更新处理器 - 场景任务需要监听消息，不检查 IsAccountBusy
		r.connectionPool.SetUpdateHandler(accountIDStr, r.createUpdateHandler(accountIDStr))
		registeredCount++
		r.logger.Info("Registered update handler for agent",
			zap.Uint64("account_id", agent.AccountID),
			zap.String("persona", agent.Persona.Name))
	}

	r.logger.Info("Agent handlers registered",
		zap.Int("registered_count", registeredCount),
		zap.Int("total_agents", len(r.scenario.Agents)))

	// 运行主循环
	duration := time.Duration(r.scenario.Duration) * time.Second
	if duration == 0 {
		duration = 10 * time.Minute // 默认10分钟
	}

	r.logger.Info("Starting main scheduling loop",
		zap.String("scenario", r.scenario.Name),
		zap.Duration("duration", duration),
		zap.Int("tick_interval_seconds", 5))

	timer := time.NewTimer(duration)
	defer timer.Stop()

	ticker := time.NewTicker(5 * time.Second) // 每5秒进行一次调度检查
	defer ticker.Stop()

	scheduleCount := 0
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Agent scenario cancelled by context",
				zap.String("scenario", r.scenario.Name),
				zap.Duration("elapsed", time.Since(startTime)),
				zap.Int("schedule_cycles", scheduleCount))
			return ctx.Err()
		case <-timer.C:
			r.logger.Info("Scenario duration reached, completing",
				zap.String("scenario", r.scenario.Name),
				zap.Duration("total_duration", time.Since(startTime)),
				zap.Int("schedule_cycles", scheduleCount))
			return nil
		case <-ticker.C:
			scheduleCount++
			r.logger.Debug("Schedule tick",
				zap.Int("cycle", scheduleCount),
				zap.Duration("elapsed", time.Since(startTime)))
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
	roll := r.rnd.Float64()
	if roll > agent.ActiveRate {
		r.logger.Debug("Agent skipped due to activity rate",
			zap.Uint64("account_id", agent.AccountID),
			zap.Float64("active_rate", agent.ActiveRate),
			zap.Float64("roll", roll))
		return // 该智能体此次不活跃
	}

	r.logger.Info("Agent selected for decision",
		zap.Uint64("account_id", agent.AccountID),
		zap.String("persona", agent.Persona.Name),
		zap.Float64("active_rate", agent.ActiveRate),
		zap.Float64("roll", roll))

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
	loopStartTime := time.Now()

	r.logger.Debug("Starting ODA loop for agent",
		zap.Uint64("account_id", agent.AccountID),
		zap.String("persona", agent.Persona.Name),
		zap.String("goal", agent.Goal))

	// 1. Observe (观察)
	// 获取最近的聊天记录
	// 这里需要通过 ConnectionPool 获取客户端并调用 API
	// 为了简化，我们假设可以通过 helper 方法获取
	history, err := r.fetchChatHistory(ctx, accountIDStr)
	if err != nil {
		r.logger.Error("Failed to fetch chat history",
			zap.Uint64("account_id", agent.AccountID),
			zap.Error(err))
		return fmt.Errorf("failed to fetch chat history: %w", err)
	}

	r.logger.Debug("Chat history fetched",
		zap.Uint64("account_id", agent.AccountID),
		zap.Int("message_count", len(history)))

	// 2. Decide (决策)
	// 构建简化的人设描述
	personaDesc := agent.Persona.Name
	if len(agent.Persona.Style) > 0 {
		personaDesc += fmt.Sprintf(" (风格: %v)", agent.Persona.Style)
	}

	decisionReq := &models.AgentDecisionRequest{
		ScenarioTopic: r.scenario.Topic,
		AgentPersona:  personaDesc,
		AgentGoal:     agent.Goal,
		ChatHistory:   history,
	}

	decision, err := r.aiService.AgentDecision(ctx, decisionReq)
	if err != nil {
		r.logger.Error("AI decision failed",
			zap.Uint64("account_id", agent.AccountID),
			zap.String("persona", agent.Persona.Name),
			zap.Error(err))
		return fmt.Errorf("AI decision failed: %w", err)
	}

	if !decision.ShouldSpeak {
		r.logger.Debug("Agent decided to stay silent",
			zap.Uint64("account_id", agent.AccountID),
			zap.String("persona", agent.Persona.Name),
			zap.String("thought", decision.Thought),
			zap.Duration("decision_time", time.Since(loopStartTime)))
		return nil
	}

	r.logger.Info("Agent decided to act",
		zap.Uint64("account_id", agent.AccountID),
		zap.String("persona", agent.Persona.Name),
		zap.String("action", decision.Action),
		zap.String("thought", decision.Thought),
		zap.Int("delay_seconds", decision.DelaySeconds))

	// 3. Act (行动)
	// 模拟延迟
	delay := time.Duration(decision.DelaySeconds) * time.Second
	if delay == 0 {
		delay = time.Duration(r.rnd.Intn(5)+2) * time.Second
	}

	// 模拟输入状态
	r.simulateTyping(ctx, accountIDStr, delay)

	// 执行发送文本消息
	return r.sendTextMessage(ctx, accountIDStr, decision.Content, 0)
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
		r.logger.Debug("Received update",
			zap.String("account_id", accountID),
			zap.String("update_type", fmt.Sprintf("%T", u)))

		switch updates := u.(type) {
		case *tg.Updates:
			r.logger.Debug("Processing Updates batch",
				zap.String("account_id", accountID),
				zap.Int("update_count", len(updates.Updates)))
			for _, update := range updates.Updates {
				r.handleUpdate(ctx, accountID, update, updates.Users)
			}
		case *tg.UpdatesCombined:
			r.logger.Debug("Processing UpdatesCombined batch",
				zap.String("account_id", accountID),
				zap.Int("update_count", len(updates.Updates)))
			for _, update := range updates.Updates {
				r.handleUpdate(ctx, accountID, update, updates.Users)
			}
		case *tg.UpdateShort:
			r.handleUpdate(ctx, accountID, updates.Update, nil)
		case *tg.UpdateShortMessage:
			r.logger.Debug("Received UpdateShortMessage",
				zap.String("account_id", accountID),
				zap.Int64("from_id", updates.UserID),
				zap.String("message", updates.Message))
		case *tg.UpdateShortChatMessage:
			r.logger.Debug("Received UpdateShortChatMessage",
				zap.String("account_id", accountID),
				zap.Int64("chat_id", updates.ChatID),
				zap.Int64("from_id", updates.FromID),
				zap.String("message", updates.Message))
		}
		return nil
	})
}

// handleUpdate 处理单个更新
func (r *AgentRunner) handleUpdate(ctx context.Context, accountID string, update tg.UpdateClass, users []tg.UserClass) {
	r.logger.Debug("Handling update",
		zap.String("account_id", accountID),
		zap.String("update_type", fmt.Sprintf("%T", update)))

	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			r.logger.Info("Received new message",
				zap.String("account_id", accountID),
				zap.Int("message_id", msg.ID),
				zap.String("content", msg.Message))
			r.processNewMessage(accountID, msg, users)
		}
	case *tg.UpdateNewChannelMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			r.logger.Info("Received new channel message",
				zap.String("account_id", accountID),
				zap.Int("message_id", msg.ID),
				zap.String("content", msg.Message))
			r.processNewMessage(accountID, msg, users)
		}
	case *tg.UpdateEditMessage:
		r.logger.Debug("Received edit message update",
			zap.String("account_id", accountID))
	case *tg.UpdateEditChannelMessage:
		r.logger.Debug("Received edit channel message update",
			zap.String("account_id", accountID))
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

	r.logger.Info("New message cached",
		zap.String("account_id", accountID),
		zap.String("sender", chatMsg.Username),
		zap.Int64("user_id", chatMsg.UserID),
		zap.String("content", chatMsg.Message),
		zap.Int("cache_size", len(r.messageCache[accountID])))
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

// ensureJoinGroup 确保账号加入目标群组
func (r *AgentRunner) ensureJoinGroup(ctx context.Context, accountID string, target string) error {
	task := &GenericTask{
		Type: "join_group",
		ExecuteFunc: func(ctx context.Context, client *gotd_telegram.Client) error {
			api := client.API()

			// 处理邀请链接
			if r.isInviteLink(target) {
				hash := r.extractInviteHash(target)
				if hash == "" {
					return fmt.Errorf("invalid invite link format")
				}
				_, err := api.MessagesImportChatInvite(ctx, hash)
				if err != nil {
					// 如果已经是成员，忽略错误
					if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
						r.logger.Debug("Already a member of the group", zap.String("account_id", accountID))
						return nil
					}
					return err
				}
				return nil
			}

			// 处理公开用户名/链接
			username := r.extractUsername(target)
			if username == "" {
				return fmt.Errorf("invalid group username or link")
			}

			// 解析用户名
			resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
				Username: username,
			})
			if err != nil {
				return fmt.Errorf("resolve username failed: %w", err)
			}

			// 加入频道/超级群
			if len(resolved.Chats) > 0 {
				if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
					// 检查是否已经是成员
					if channel.Left {
						// 尝试加入
						_, err = api.ChannelsJoinChannel(ctx, &tg.InputChannel{
							ChannelID:  channel.ID,
							AccessHash: channel.AccessHash,
						})
						if err != nil {
							// 如果已经是成员，忽略错误
							if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
								return nil
							}
							return err
						}
					}
					return nil
				}
				return fmt.Errorf("target is not a channel or supergroup")
			}

			return fmt.Errorf("group not found")
		},
	}
	return r.connectionPool.ExecuteTask(accountID, task)
}

// isInviteLink 检查是否为邀请链接
func (r *AgentRunner) isInviteLink(input string) bool {
	return strings.Contains(input, "joinchat") || strings.Contains(input, "/+")
}

// extractInviteHash 提取邀请哈希
func (r *AgentRunner) extractInviteHash(input string) string {
	// 处理 https://t.me/joinchat/Hash
	if idx := strings.Index(input, "joinchat/"); idx != -1 {
		return input[idx+9:]
	}
	// 处理 https://t.me/+Hash
	if idx := strings.Index(input, "/+"); idx != -1 {
		return input[idx+2:]
	}
	return ""
}

// extractUsername 提取用户名
func (r *AgentRunner) extractUsername(input string) string {
	// 移除 https://t.me/ 或 t.me/
	input = strings.TrimPrefix(input, "https://")
	input = strings.TrimPrefix(input, "http://")
	input = strings.TrimPrefix(input, "t.me/")
	// 移除 @ 前缀
	input = strings.TrimPrefix(input, "@")
	return input
}
