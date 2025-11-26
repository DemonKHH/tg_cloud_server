package telegram

import (
	"context"
	"fmt"
	"time"

	"tg_cloud_server/internal/models"

	gotd_telegram "github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

// TaskInterface 任务执行器接口
type TaskInterface interface {
	Execute(ctx context.Context, api *tg.Client) error
	GetType() string
}

// AdvancedTaskInterface 高级任务执行器接口 (支持完整Client)
type AdvancedTaskInterface interface {
	TaskInterface
	ExecuteAdvanced(ctx context.Context, client *gotd_telegram.Client) error
}

// AccountCheckTask 账号检查任务
type AccountCheckTask struct {
	task *models.Task
}

// NewAccountCheckTask 创建账号检查任务
func NewAccountCheckTask(task *models.Task) *AccountCheckTask {
	return &AccountCheckTask{task: task}
}

// Execute 执行账号检查
func (t *AccountCheckTask) Execute(ctx context.Context, api *tg.Client) error {
	// 初始化检查结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	checkResults := make(map[string]interface{})
	checkScore := 100.0
	var issues []string
	var suggestions []string

	// 1. 基本账号信息检查
	_, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		checkScore -= 50
		issues = append(issues, "无法获取账号基本信息")
		suggestions = append(suggestions, "检查账号登录状态")
		checkResults["basic_info_check"] = "failed"
		checkResults["error"] = err.Error()
	} else {
		checkResults["basic_info_check"] = "passed"
		checkResults["user_retrieved"] = true
	}

	// 2. 连接状态检查
	_, err = api.HelpGetConfig(ctx)
	if err != nil {
		checkScore -= 30
		issues = append(issues, "Telegram服务连接异常")
		suggestions = append(suggestions, "检查网络连接和代理设置")
		checkResults["connection_check"] = "failed"
	} else {
		checkResults["connection_check"] = "passed"
	}

	// 3. 对话列表检查 (检查账号是否能正常获取数据)
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit: 5,
	})
	if err != nil {
		checkScore -= 20
		issues = append(issues, "无法获取对话列表")
		suggestions = append(suggestions, "检查账号是否被限制")
		checkResults["dialogs_check"] = "failed"
	} else {
		checkResults["dialogs_check"] = "passed"
		if messagesDialogs, ok := dialogs.(*tg.MessagesDialogs); ok {
			checkResults["dialogs_count"] = len(messagesDialogs.Dialogs)
		}
	}

	// 4. 发送能力检查 (尝试获取应用配置)
	_, err = api.HelpGetAppConfig(ctx, 0)
	if err != nil {
		checkResults["limits_check"] = "skipped"
	} else {
		checkResults["limits_check"] = "passed"
		checkResults["config_retrieved"] = true
	}

	// 5. 账号状态评估
	if checkScore >= 90 {
		checkResults["account_status"] = "excellent"
	} else if checkScore >= 70 {
		checkResults["account_status"] = "good"
	} else if checkScore >= 50 {
		checkResults["account_status"] = "warning"
	} else {
		checkResults["account_status"] = "critical"
	}

	// 更新任务结果
	t.task.Result["check_score"] = checkScore
	t.task.Result["issues"] = issues
	t.task.Result["suggestions"] = suggestions
	t.task.Result["check_results"] = checkResults
	t.task.Result["check_time"] = time.Now().Unix()
	t.task.Result["status"] = "completed"

	return nil
}

// GetType 获取任务类型
func (t *AccountCheckTask) GetType() string {
	return "account_check"
}

// PrivateMessageTask 私信任务
type PrivateMessageTask struct {
	task *models.Task
}

// NewPrivateMessageTask 创建私信任务
func NewPrivateMessageTask(task *models.Task) *PrivateMessageTask {
	return &PrivateMessageTask{task: task}
}

// Execute 执行私信发送
func (t *PrivateMessageTask) Execute(ctx context.Context, api *tg.Client) error {
	config := t.task.Config

	// 验证配置完整性
	if config == nil {
		return fmt.Errorf("task config is nil")
	}

	// 获取目标用户列表
	targets, ok := config["targets"].([]interface{})
	if !ok || len(targets) == 0 {
		return fmt.Errorf("invalid or empty targets configuration")
	}

	// 获取消息内容
	message, ok := config["message"].(string)
	if !ok || message == "" {
		return fmt.Errorf("invalid or empty message configuration")
	}

	// 添加调试日志
	fmt.Printf("[PrivateMessageTask] Task ID: %d, Targets: %d, Message length: %d\n",
		t.task.ID, len(targets), len(message))

	// 获取发送间隔 (防止频繁发送被限制)
	intervalSec := 2 // 默认2秒间隔
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	sentCount := 0
	failedCount := 0
	var errors []string
	var sentTargets []string
	targetResults := make(map[string]interface{}) // 记录每个目标的详细结果

	// 发送私信给每个目标用户
	for i, target := range targets {
		// 添加发送间隔（除了第一个消息）
		if i > 0 && intervalSec > 0 {
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}

		username, ok := target.(string)
		if !ok {
			errorMsg := fmt.Sprintf("invalid target format: %v", target)
			errors = append(errors, errorMsg)
			targetResults[fmt.Sprintf("target_%d", i+1)] = map[string]interface{}{
				"target": target,
				"status": "failed",
				"error":  errorMsg,
			}
			failedCount++
			continue
		}

		// 添加调试日志
		fmt.Printf("[PrivateMessageTask] Sending to %s (%d/%d)\n", username, i+1, len(targets))

		// 记录发送开始时间
		sendStartTime := time.Now()

		// 尝试通过用户名解析
		err := t.sendPrivateMessage(ctx, api, username, message)
		sendDuration := time.Since(sendStartTime)

		if err != nil {
			fmt.Printf("[PrivateMessageTask] Failed to send to %s: %v\n", username, err)
			errorMsg := fmt.Sprintf("failed to send to %s: %v", username, err)
			errors = append(errors, errorMsg)
			targetResults[username] = map[string]interface{}{
				"status":   "failed",
				"error":    err.Error(),
				"duration": sendDuration.String(),
			}
			failedCount++
		} else {
			fmt.Printf("[PrivateMessageTask] Successfully sent to %s\n", username)
			sentCount++
			sentTargets = append(sentTargets, username)
			targetResults[username] = map[string]interface{}{
				"status":   "success",
				"duration": sendDuration.String(),
			}
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["sent_count"] = sentCount
	t.task.Result["failed_count"] = failedCount
	t.task.Result["errors"] = errors
	t.task.Result["sent_targets"] = sentTargets
	t.task.Result["target_results"] = targetResults // 添加每个目标的详细结果
	t.task.Result["total_targets"] = len(targets)
	t.task.Result["success_rate"] = float64(sentCount) / float64(len(targets))
	t.task.Result["send_time"] = time.Now().Unix()

	// 添加调试日志
	fmt.Printf("[PrivateMessageTask] Execution completed: sent=%d, failed=%d, total=%d\n",
		sentCount, failedCount, len(targets))

	return nil
}

// sendPrivateMessage 发送私信给指定用户
func (t *PrivateMessageTask) sendPrivateMessage(ctx context.Context, api *tg.Client, username, message string) error {
	// 移除用户名前的@符号（如果有的话）
	cleanUsername := username
	if len(username) > 0 && username[0] == '@' {
		cleanUsername = username[1:]
	}

	// 通过用户名解析
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: cleanUsername,
	})
	if err != nil {
		return fmt.Errorf("username not found: %w", err)
	}

	// 从解析结果中获取用户信息
	if len(resolved.Users) > 0 {
		if user, ok := resolved.Users[0].(*tg.User); ok {
			inputPeer := &tg.InputPeerUser{
				UserID:     user.ID,
				AccessHash: user.AccessHash,
			}

			// 发送消息
			_, err = api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
				Peer:     inputPeer,
				Message:  message,
				RandomID: time.Now().UnixNano(), // 防止重复消息
			})

			return err
		}
	}

	return fmt.Errorf("user not found: %s", username)
}

// GetType 获取任务类型
func (t *PrivateMessageTask) GetType() string {
	return "private_message"
}

// BroadcastTask 群发任务
type BroadcastTask struct {
	task *models.Task
}

// NewBroadcastTask 创建群发任务
func NewBroadcastTask(task *models.Task) *BroadcastTask {
	return &BroadcastTask{task: task}
}

// Execute 执行群发消息
func (t *BroadcastTask) Execute(ctx context.Context, api *tg.Client) error {
	config := t.task.Config

	// 验证配置完整性
	if config == nil {
		return fmt.Errorf("task config is nil")
	}

	// 获取目标群组列表 (支持群组ID或群组用户名)
	groups, ok := config["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		return fmt.Errorf("invalid or empty groups configuration")
	}

	// 获取消息内容
	message, ok := config["message"].(string)
	if !ok || message == "" {
		return fmt.Errorf("invalid or empty message configuration")
	}

	// 获取发送间隔 (防止被限制)
	intervalSec := 3 // 默认3秒间隔，群发更谨慎
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	sentCount := 0
	failedCount := 0
	var errors []string
	var sentGroups []string

	// 发送消息到每个群组
	for i, group := range groups {
		// 添加发送间隔（除了第一个消息）
		if i > 0 && intervalSec > 0 {
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}

		err := t.sendBroadcastMessage(ctx, api, group, message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to send to group %v: %v", group, err))
			failedCount++
		} else {
			sentCount++
			sentGroups = append(sentGroups, fmt.Sprintf("%v", group))
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["sent_count"] = sentCount
	t.task.Result["failed_count"] = failedCount
	t.task.Result["errors"] = errors
	t.task.Result["sent_groups"] = sentGroups
	t.task.Result["total_groups"] = len(groups)
	t.task.Result["success_rate"] = float64(sentCount) / float64(len(groups))
	t.task.Result["send_time"] = time.Now().Unix()

	return nil
}

// sendBroadcastMessage 发送群发消息到指定群组
func (t *BroadcastTask) sendBroadcastMessage(ctx context.Context, api *tg.Client, group interface{}, message string) error {
	var inputPeer tg.InputPeerClass

	switch v := group.(type) {
	case int64:
		// 如果是数字ID，尝试作为ChatID
		inputPeer = &tg.InputPeerChat{ChatID: v}
	case float64:
		// JSON解码数字可能是float64
		inputPeer = &tg.InputPeerChat{ChatID: int64(v)}
	case string:
		// 如果是字符串，尝试解析为群组用户名
		cleanGroupname := v
		if len(v) > 0 && v[0] == '@' {
			cleanGroupname = v[1:]
		}

		// 移除可能的链接前缀
		if len(cleanGroupname) > 13 && cleanGroupname[:13] == "https://t.me/" {
			cleanGroupname = cleanGroupname[13:]
		} else if len(cleanGroupname) > 5 && cleanGroupname[:5] == "t.me/" {
			cleanGroupname = cleanGroupname[5:]
		}

		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: cleanGroupname,
		})
		if err != nil {
			return fmt.Errorf("group not found: %w", err)
		}

		// 从解析结果中获取群组信息
		if len(resolved.Chats) > 0 {
			if chat, ok := resolved.Chats[0].(*tg.Chat); ok {
				inputPeer = &tg.InputPeerChat{ChatID: chat.ID}
			} else if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
				inputPeer = &tg.InputPeerChannel{
					ChannelID:  channel.ID,
					AccessHash: channel.AccessHash,
				}
			} else {
				return fmt.Errorf("unsupported chat type")
			}
		} else {
			return fmt.Errorf("group not found: %s", cleanGroupname)
		}
	default:
		return fmt.Errorf("unsupported group identifier type: %T", group)
	}

	// 发送消息
	_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     inputPeer,
		Message:  message,
		RandomID: time.Now().UnixNano(),
	})

	return err
}

// GetType 获取任务类型
func (t *BroadcastTask) GetType() string {
	return "broadcast"
}

// VerifyCodeTask 验证码接收任务
type VerifyCodeTask struct {
	task *models.Task
}

// NewVerifyCodeTask 创建验证码接收任务
func NewVerifyCodeTask(task *models.Task) *VerifyCodeTask {
	return &VerifyCodeTask{task: task}
}

// Execute 执行验证码接收
func (t *VerifyCodeTask) Execute(ctx context.Context, api *tg.Client) error {
	config := t.task.Config

	// 验证配置完整性
	if config == nil {
		return fmt.Errorf("task config is nil")
	}

	// 获取监听的发送者列表 (可以是官方验证服务、特定用户等)
	senders := []string{"777000", "Telegram"} // 默认Telegram官方
	if configSenders, exists := config["senders"]; exists {
		if sendersSlice, ok := configSenders.([]interface{}); ok {
			senders = make([]string, 0, len(sendersSlice))
			for _, sender := range sendersSlice {
				if senderStr, ok := sender.(string); ok {
					senders = append(senders, senderStr)
				}
			}
		}
	}

	// 获取超时时间
	timeoutSec := 300 // 默认5分钟超时
	if timeout, exists := config["timeout_seconds"]; exists {
		if timeoutFloat, ok := timeout.(float64); ok && timeoutFloat > 0 {
			timeoutSec = int(timeoutFloat)
		}
	}

	// 限制超时时间范围
	if timeoutSec < 30 {
		timeoutSec = 30 // 最少30秒
	} else if timeoutSec > 600 {
		timeoutSec = 600 // 最多10分钟
	}

	startTime := time.Now()
	var verifyCode string
	var receivedAt time.Time
	var senderInfo string

	// 轮询检查新消息
	for time.Since(startTime) < time.Duration(timeoutSec)*time.Second {
		// 获取最新对话
		dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			Limit: 20,
		})
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		// 检查每个对话的最新消息
		code, sender, receivedTime, found := t.searchVerifyCode(dialogs, senders, startTime)
		if found {
			verifyCode = code
			senderInfo = sender
			receivedAt = receivedTime
			break
		}

		// 等待2秒后再次检查
		time.Sleep(2 * time.Second)
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	if verifyCode != "" {
		t.task.Result["verify_code"] = verifyCode
		t.task.Result["sender"] = senderInfo
		t.task.Result["received_at"] = receivedAt.Unix()
		t.task.Result["status"] = "received"
	} else {
		t.task.Result["verify_code"] = ""
		t.task.Result["status"] = "timeout"
		t.task.Result["error"] = "verification code not received within timeout"
	}

	t.task.Result["timeout_seconds"] = timeoutSec
	t.task.Result["actual_wait_seconds"] = int(time.Since(startTime).Seconds())

	return nil
}

// searchVerifyCode 在对话中搜索验证码
func (t *VerifyCodeTask) searchVerifyCode(dialogs tg.MessagesDialogsClass, senders []string, startTime time.Time) (code, sender string, receivedTime time.Time, found bool) {
	if messagesDialogs, ok := dialogs.(*tg.MessagesDialogs); ok {
		for _, message := range messagesDialogs.Messages {
			if msg, ok := message.(*tg.Message); ok {
				// 检查消息时间是否在任务开始后
				msgTime := time.Unix(int64(msg.Date), 0)
				if msgTime.Before(startTime) {
					continue
				}

				// 检查发送者
				var msgSender string
				if msg.FromID != nil {
					if peerUser, ok := msg.FromID.(*tg.PeerUser); ok {
						msgSender = fmt.Sprintf("%d", peerUser.UserID)
					}
				} else {
					msgSender = "777000" // Telegram系统消息
				}

				// 验证发送者是否在白名单中
				senderMatched := false
				for _, allowedSender := range senders {
					if msgSender == allowedSender {
						senderMatched = true
						break
					}
				}

				if !senderMatched {
					continue
				}

				// 解析验证码
				if extractedCode := t.extractVerificationCode(msg.Message); extractedCode != "" {
					return extractedCode, msgSender, msgTime, true
				}
			}
		}
	}

	return "", "", time.Time{}, false
}

// extractVerificationCode 从消息文本中提取验证码
func (t *VerifyCodeTask) extractVerificationCode(message string) string {
	// 常见的验证码模式
	patterns := []string{
		"code", "verification", "verify", "login", "telegram",
		"验证码", "验证", "登录", "代码",
	}

	// 简单的数字提取逻辑 (4-8位数字)
	var digits []rune
	for _, char := range message {
		if char >= '0' && char <= '9' {
			digits = append(digits, char)
		}
	}

	// 检查是否包含验证码关键词
	messageContainsPattern := false
	for _, pattern := range patterns {
		if t.containsIgnoreCase(message, pattern) {
			messageContainsPattern = true
			break
		}
	}

	// 如果包含关键词且数字长度合适
	if messageContainsPattern && len(digits) >= 4 && len(digits) <= 8 {
		return string(digits)
	}

	return ""
}

// containsIgnoreCase 不区分大小写的包含检查
func (t *VerifyCodeTask) containsIgnoreCase(text, pattern string) bool {
	textLower := t.toLowerCase(text)
	patternLower := t.toLowerCase(pattern)

	return t.contains(textLower, patternLower)
}

// toLowerCase 转换为小写
func (t *VerifyCodeTask) toLowerCase(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// contains 检查字符串是否包含子字符串
func (t *VerifyCodeTask) contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetType 获取任务类型
func (t *VerifyCodeTask) GetType() string {
	return "verify_code"
}

// GroupChatTask AI炒群任务
type GroupChatTask struct {
	task *models.Task
}

// NewGroupChatTask 创建AI炒群任务
func NewGroupChatTask(task *models.Task) *GroupChatTask {
	return &GroupChatTask{task: task}
}

// Execute 执行AI炒群
func (t *GroupChatTask) Execute(ctx context.Context, api *tg.Client) error {
	config := t.task.Config

	// 验证配置完整性
	if config == nil {
		return fmt.Errorf("task config is nil")
	}

	// 获取目标群组（支持ID和用户名）
	var inputPeer tg.InputPeerClass
	if groupID, ok := config["group_id"].(float64); ok && groupID > 0 {
		inputPeer = &tg.InputPeerChat{ChatID: int64(groupID)}
	} else if groupName, ok := config["group_name"].(string); ok && groupName != "" {
		// 解析群组用户名
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: groupName,
		})
		if err != nil {
			return fmt.Errorf("failed to resolve group: %w", err)
		}
		if len(resolved.Chats) > 0 {
			if chat, ok := resolved.Chats[0].(*tg.Chat); ok {
				inputPeer = &tg.InputPeerChat{ChatID: chat.ID}
			} else if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
				inputPeer = &tg.InputPeerChannel{
					ChannelID:  channel.ID,
					AccessHash: channel.AccessHash,
				}
			}
		}
	} else {
		return fmt.Errorf("missing group_id or group_name configuration")
	}

	// 获取AI配置
	aiConfig, ok := config["ai_config"].(map[string]interface{})
	if !ok {
		// 使用默认AI配置
		aiConfig = map[string]interface{}{
			"personality":   "friendly",
			"response_rate": 0.3,
			"keywords":      []string{"hello", "hi", "question"},
		}
	}

	// 获取监控时长
	monitorDuration := 300 // 默认5分钟
	if duration, exists := config["monitor_duration_seconds"]; exists {
		if durationFloat, ok := duration.(float64); ok {
			monitorDuration = int(durationFloat)
		}
	}

	responseSent := 0
	messagesProcessed := 0

	// 获取群组最新消息作为初始检查
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		Limit: 5,
	})
	if err != nil {
		return fmt.Errorf("failed to get chat history: %w", err)
	}

	// 分析群聊上下文并可能发送回复
	if messages, ok := history.(*tg.MessagesMessages); ok {
		for _, msg := range messages.Messages {
			if message, ok := msg.(*tg.Message); ok {
				messagesProcessed++

				// 简单的回复逻辑 - 如果消息包含关键词且随机数允许
				if t.shouldRespondSimple(message, aiConfig) {
					response := t.generateSimpleAIResponse(message, aiConfig)
					if response != "" {
						_, err = api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
							Peer:     inputPeer,
							Message:  response,
							RandomID: time.Now().UnixNano(),
						})
						if err == nil {
							responseSent++
						}
						break // 只发送一个回复
					}
				}
			}
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["messages_processed"] = messagesProcessed
	t.task.Result["responses_sent"] = responseSent
	t.task.Result["monitor_duration"] = monitorDuration
	t.task.Result["completion_time"] = time.Now().Unix()

	return nil
}

// shouldRespondSimple 简单的回复决策逻辑
func (t *GroupChatTask) shouldRespondSimple(msg *tg.Message, aiConfig map[string]interface{}) bool {
	// 获取回复概率
	responseRate := 0.3 // 默认30%
	if rate, exists := aiConfig["response_rate"]; exists {
		if rateFloat, ok := rate.(float64); ok {
			responseRate = rateFloat
		}
	}

	// 基础概率检查
	if t.simpleRandom() > responseRate {
		return false
	}

	// 检查关键词
	keywords, exists := aiConfig["keywords"].([]interface{})
	if exists && len(keywords) > 0 {
		for _, keyword := range keywords {
			if keywordStr, ok := keyword.(string); ok {
				if t.containsIgnoreCase(msg.Message, keywordStr) {
					return true
				}
			}
		}
		// 如果有关键词配置但都不匹配，降低概率
		return t.simpleRandom() < 0.1
	}

	return true
}

// generateSimpleAIResponse 生成简单的AI回复
func (t *GroupChatTask) generateSimpleAIResponse(msg *tg.Message, aiConfig map[string]interface{}) string {
	personality := "friendly"
	if p, exists := aiConfig["personality"]; exists {
		if pStr, ok := p.(string); ok {
			personality = pStr
		}
	}

	msgLower := t.toLowerCase(msg.Message)

	// 根据消息内容选择回复
	if t.contains(msgLower, "hello") || t.contains(msgLower, "hi") || t.contains(msgLower, "你好") {
		responses := []string{"Hello there! 👋", "Hi! How's everyone? 😊", "Hey! 🙋‍♂️"}
		return responses[t.simpleRandomInt(len(responses))]
	}

	if t.contains(msgLower, "thank") || t.contains(msgLower, "谢谢") || t.contains(msgLower, "thx") {
		responses := []string{"You're welcome! 😊", "No problem! 👍", "Happy to help! 🤝"}
		return responses[t.simpleRandomInt(len(responses))]
	}

	if t.contains(msgLower, "?") || t.contains(msgLower, "？") || t.contains(msgLower, "问") {
		responses := []string{"That's a good question! 🤔", "Interesting point! 💭", "Let me think about that... 🧠"}
		return responses[t.simpleRandomInt(len(responses))]
	}

	// 根据个性选择默认回复
	switch personality {
	case "friendly":
		responses := []string{"I agree! 👌", "That's so true! ✨", "Absolutely! 💯", "Makes sense! 🎯"}
		return responses[t.simpleRandomInt(len(responses))]
	case "professional":
		responses := []string{"I concur.", "That's correct.", "Understood.", "Good point."}
		return responses[t.simpleRandomInt(len(responses))]
	default:
		responses := []string{"👍", "😊", "Indeed", "Right!", "Cool! 😎"}
		return responses[t.simpleRandomInt(len(responses))]
	}
}

// 简单的随机数函数
func (t *GroupChatTask) simpleRandom() float64 {
	return float64((time.Now().UnixNano() % 100)) / 100.0
}

func (t *GroupChatTask) simpleRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	return int(time.Now().UnixNano() % int64(max))
}

// containsIgnoreCase 不区分大小写的包含检查 (GroupChatTask版本)
func (t *GroupChatTask) containsIgnoreCase(text, pattern string) bool {
	textLower := t.toLowerCase(text)
	patternLower := t.toLowerCase(pattern)

	return t.contains(textLower, patternLower)
}

// toLowerCase 转换为小写 (GroupChatTask版本)
func (t *GroupChatTask) toLowerCase(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// contains 检查字符串是否包含子字符串 (GroupChatTask版本)
func (t *GroupChatTask) contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetType 获取任务类型
func (t *GroupChatTask) GetType() string {
	return "group_chat"
}
