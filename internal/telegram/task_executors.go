package telegram

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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

	// 初始化日志
	var logs []string
	addLog := func(msg string) {
		logEntry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		logs = append(logs, logEntry)
		t.task.Result["logs"] = logs
	}

	addLog("开始执行账号检查任务...")

	checkResults := make(map[string]interface{})
	checkScore := 100.0
	var issues []string
	var suggestions []string

	// 1. 基本账号信息检查
	addLog("正在获取基本账号信息...")
	user, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		checkScore -= 50
		issues = append(issues, "无法获取账号基本信息")
		suggestions = append(suggestions, "检查账号登录状态")
		checkResults["basic_info_check"] = "failed"
		checkResults["error"] = err.Error()
		addLog(fmt.Sprintf("基本信息获取失败: %v", err))
	} else {
		checkResults["basic_info_check"] = "passed"
		checkResults["user_retrieved"] = true
		if len(user.Users) > 0 {
			if u, ok := user.Users[0].(*tg.User); ok {
				addLog(fmt.Sprintf("基本信息获取成功: %s %s (ID: %d)", u.FirstName, u.LastName, u.ID))
			}
		}
	}

	// 2. 连接状态检查
	addLog("正在检查连接状态...")
	_, err = api.HelpGetConfig(ctx)
	if err != nil {
		checkScore -= 30
		issues = append(issues, "Telegram服务连接异常")
		suggestions = append(suggestions, "检查网络连接和代理设置")
		checkResults["connection_check"] = "failed"
		addLog(fmt.Sprintf("连接状态异常: %v", err))
	} else {
		checkResults["connection_check"] = "passed"
		addLog("连接状态正常")
	}

	// 3. 对话列表检查 (检查账号是否能正常获取数据)
	addLog("正在检查对话列表...")
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit: 5,
	})
	if err != nil {
		checkScore -= 20
		issues = append(issues, "无法获取对话列表")
		suggestions = append(suggestions, "检查账号是否被限制")
		checkResults["dialogs_check"] = "failed"
		addLog(fmt.Sprintf("无法获取对话列表: %v", err))
	} else {
		checkResults["dialogs_check"] = "passed"
		if messagesDialogs, ok := dialogs.(*tg.MessagesDialogs); ok {
			count := len(messagesDialogs.Dialogs)
			checkResults["dialogs_count"] = count
			addLog(fmt.Sprintf("对话列表获取成功，最近对话数: %d", count))
		} else if messagesDialogsSlice, ok := dialogs.(*tg.MessagesDialogsSlice); ok {
			count := len(messagesDialogsSlice.Dialogs)
			checkResults["dialogs_count"] = count
			addLog(fmt.Sprintf("对话列表获取成功，最近对话数: %d", count))
		}
	}

	// 4. 发送能力检查 (尝试获取应用配置)
	addLog("正在检查应用配置...")
	_, err = api.HelpGetAppConfig(ctx, 0)
	if err != nil {
		checkResults["limits_check"] = "skipped"
		addLog("应用配置获取失败 (跳过)")
	} else {
		checkResults["limits_check"] = "passed"
		checkResults["config_retrieved"] = true
		addLog("应用配置获取成功")
	}

	// 5. 2FA 检查 (可选)
	if check2FA, ok := t.task.Config["check_2fa"].(bool); ok && check2FA {
		addLog("正在检查 2FA 状态...")
		password, err := api.AccountGetPassword(ctx)
		if err != nil {
			checkScore -= 10
			issues = append(issues, fmt.Sprintf("无法获取2FA状态: %v", err))
			checkResults["2fa_check"] = "failed"
			addLog(fmt.Sprintf("2FA 状态获取失败: %v", err))
		} else {
			has2FA := password.HasPassword
			checkResults["has_2fa"] = has2FA
			checkResults["2fa_check"] = "passed"

			if has2FA {
				addLog("账号已开启 2FA")
				// 如果开启了2FA，检查密码是否正确
				twoFAPassword, _ := t.task.Config["two_fa_password"].(string)
				checkResults["two_fa_password"] = twoFAPassword

				if twoFAPassword != "" {
					checkResults["is_2fa_correct"] = "unchecked"
					suggestions = append(suggestions, "账号已开启2FA，请确保记录了正确的密码")
					addLog("已配置 2FA 密码 (未验证正确性)")
				} else {
					checkScore -= 10
					issues = append(issues, "账号开启了2FA但未提供密码")
					suggestions = append(suggestions, "请补充2FA密码")
					checkResults["is_2fa_correct"] = "missing"
					addLog("警告: 账号开启了 2FA 但未提供密码")
				}
			} else {
				suggestions = append(suggestions, "建议开启2FA以提高账号安全性")
				addLog("账号未开启 2FA")
			}
		}
	}

	// 6. SpamBot 检查 (可选)
	if checkSpamBot, ok := t.task.Config["check_spam_bot"].(bool); ok && checkSpamBot {
		addLog("正在执行 SpamBot 检查...")
		messageText, err := t.checkSpamBot(ctx, api)
		if err != nil {
			checkScore -= 20
			issues = append(issues, fmt.Sprintf("SpamBot检查失败: %v", err))
			checkResults["spam_bot_check"] = "failed"
			addLog(fmt.Sprintf("SpamBot 检查失败: %v", err))
		} else {
			checkResults["spam_bot_check"] = "passed"
			checkResults["spambot_response"] = messageText
			addLog("SpamBot 响应获取成功")

			// 转换为小写以便匹配
			messageTextLower := strings.ToLower(messageText)

			// 检查双向限制
			bidirectionalKeywords := []string{
				"restricted from",
				"can't message people",
				"cannot message people",
				"can't send messages",
				"cannot send messages",
				"messaging strangers",
				"marked as spam",
			}

			isBidirectional := false
			for _, keyword := range bidirectionalKeywords {
				if strings.Contains(messageTextLower, keyword) {
					isBidirectional = true
					break
				}
			}
			checkResults["is_bidirectional"] = isBidirectional

			// 检查冻结状态
			frozenKeywords := []string{
				"account was blocked",
				"account has been blocked",
				"blocked for violations",
				"permanently blocked",
				"blocked.{1,20}cannot be restored", // Go的strings.Contains不支持正则，这里简化处理，稍后用正则
				"account is limited",
				"permanently limited",
				"violated the terms of service",
			}

			// 使用正则进行更精确的匹配
			isFrozen := false
			for _, keyword := range frozenKeywords {
				matched, _ := regexp.MatchString(keyword, messageTextLower)
				if matched {
					isFrozen = true
					break
				}
			}
			checkResults["is_frozen"] = isFrozen

			if isFrozen {
				// 提取冻结结束时间
				re := regexp.MustCompile(`limited until ([^\.]+)`)
				matches := re.FindStringSubmatch(messageText)
				if len(matches) > 1 {
					checkResults["frozen_until"] = matches[1]
				}
			}

			// 根据检查结果更新建议和分数
			if isFrozen {
				checkScore = 0 // 冻结账号分数为0
				issues = append(issues, "账号已被冻结或严重受限")
				suggestions = append(suggestions, "建议将账号状态设置为: 冻结 (Frozen)")
				checkResults["suggested_status"] = "frozen"
				addLog("检测结果: 账号已被冻结")
			} else if isBidirectional {
				checkScore -= 50
				issues = append(issues, "账号处于双向限制状态")
				suggestions = append(suggestions, "建议将账号状态设置为: 双向 (Two-way)")
				checkResults["suggested_status"] = "two_way"
				addLog("检测结果: 账号处于双向限制状态")
			} else if strings.Contains(messageTextLower, "good news, no limits are currently applied") {
				// 账号正常
				addLog("检测结果: 账号状态正常")
			} else {
				// 其他未知限制
				checkScore -= 20
				issues = append(issues, "账号存在未知限制")
				checkResults["unknown_limits"] = true
				addLog("检测结果: 账号存在未知限制")
			}
		}
	}

	// 6. 账号状态评估
	if checkScore >= 90 {
		checkResults["account_status"] = "excellent"
	} else if checkScore >= 70 {
		checkResults["account_status"] = "good"
	} else if checkScore >= 50 {
		checkResults["account_status"] = "warning"
	} else {
		checkResults["account_status"] = "critical"
	}

	addLog(fmt.Sprintf("检查完成，综合评分: %.0f", checkScore))

	// 更新任务结果
	t.task.Result["check_score"] = checkScore
	t.task.Result["issues"] = issues
	t.task.Result["suggestions"] = suggestions
	t.task.Result["check_results"] = checkResults
	t.task.Result["check_time"] = time.Now().Unix()
	t.task.Result["status"] = "completed"

	// 将关键结果提升到顶层，以便 TaskScheduler 处理
	if val, ok := checkResults["suggested_status"]; ok {
		t.task.Result["suggested_status"] = val
	}
	if val, ok := checkResults["has_2fa"]; ok {
		t.task.Result["has_2fa"] = val
	}
	if val, ok := checkResults["two_fa_password"]; ok {
		t.task.Result["two_fa_password"] = val
	}
	if val, ok := checkResults["frozen_until"]; ok {
		t.task.Result["frozen_until"] = val
	}
	if val, ok := checkResults["2fa_check"]; ok {
		t.task.Result["2fa_check"] = val
	}
	if val, ok := checkResults["is_2fa_correct"]; ok {
		t.task.Result["is_2fa_correct"] = val
	}
	if val, ok := checkResults["spam_bot_check"]; ok {
		t.task.Result["spam_bot_check"] = val
	}
	if val, ok := checkResults["is_frozen"]; ok {
		t.task.Result["is_frozen"] = val
	}
	if val, ok := checkResults["is_bidirectional"]; ok {
		t.task.Result["is_bidirectional"] = val
	}

	return nil
}

// checkSpamBot 检查 SpamBot 状态
func (t *AccountCheckTask) checkSpamBot(ctx context.Context, api *tg.Client) (string, error) {
	// 解析 SpamBot
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: "SpamBot",
	})
	if err != nil {
		return "", fmt.Errorf("failed to resolve SpamBot: %w", err)
	}

	var inputPeer tg.InputPeerClass
	var botInputUser *tg.InputUser
	if len(resolved.Users) > 0 {
		if user, ok := resolved.Users[0].(*tg.User); ok {
			inputPeer = &tg.InputPeerUser{
				UserID:     user.ID,
				AccessHash: user.AccessHash,
			}
			botInputUser = &tg.InputUser{
				UserID:     user.ID,
				AccessHash: user.AccessHash,
			}
		}
	}

	if inputPeer == nil || botInputUser == nil {
		return "", fmt.Errorf("SpamBot user not found")
	}

	// 发送 /start
	_, err = api.MessagesStartBot(ctx, &tg.MessagesStartBotRequest{
		Bot:        botInputUser,
		Peer:       inputPeer,
		RandomID:   time.Now().UnixNano(),
		StartParam: "",
	})
	if err != nil {
		// 如果 StartBot 失败，尝试直接发送消息
		_, err = api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:     inputPeer,
			Message:  "/start",
			RandomID: time.Now().UnixNano(),
		})
		if err != nil {
			return "", fmt.Errorf("failed to start SpamBot: %w", err)
		}
	}

	// 等待响应
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for SpamBot response")
		case <-ticker.C:
			history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
				Peer:  inputPeer,
				Limit: 1,
			})
			if err != nil {
				continue
			}

			if messages, ok := history.(*tg.MessagesMessages); ok {
				if len(messages.Messages) > 0 {
					if msg, ok := messages.Messages[0].(*tg.Message); ok {
						// 检查是否是最近的消息 (例如最近1分钟内)
						if time.Since(time.Unix(int64(msg.Date), 0)) < 1*time.Minute {
							return msg.Message, nil
						}
					}
				}
			} else if messagesSlice, ok := history.(*tg.MessagesMessagesSlice); ok {
				if len(messagesSlice.Messages) > 0 {
					if msg, ok := messagesSlice.Messages[0].(*tg.Message); ok {
						if time.Since(time.Unix(int64(msg.Date), 0)) < 1*time.Minute {
							return msg.Message, nil
						}
					}
				}
			}
		}
	}
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

	// 初始化日志
	var logs []string
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	addLog := func(msg string) {
		logEntry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		logs = append(logs, logEntry)
		t.task.Result["logs"] = logs
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

	// 获取发送间隔 (防止频繁发送被限制)
	intervalSec := 2 // 默认2秒间隔
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	addLog(fmt.Sprintf("开始执行私信任务，目标用户数: %d，间隔: %d秒", len(targets), intervalSec))

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
			addLog(fmt.Sprintf("目标格式错误: %v", target))
			continue
		}

		// 尝试通过用户名解析
		sendStartTime := time.Now()
		err := t.sendPrivateMessage(ctx, api, username, message)
		sendDuration := time.Since(sendStartTime)

		if err != nil {
			errorMsg := fmt.Sprintf("failed to send to %s: %v", username, err)
			errors = append(errors, errorMsg)
			targetResults[username] = map[string]interface{}{
				"status":   "failed",
				"error":    err.Error(),
				"duration": sendDuration.String(),
			}
			failedCount++
			addLog(fmt.Sprintf("发送失败 [%s]: %v", username, err))
		} else {
			sentCount++
			sentTargets = append(sentTargets, username)
			targetResults[username] = map[string]interface{}{
				"status":   "success",
				"duration": sendDuration.String(),
			}
			addLog(fmt.Sprintf("发送成功: %s", username))
		}
	}

	// 更新任务结果
	t.task.Result["sent_count"] = sentCount
	t.task.Result["failed_count"] = failedCount
	t.task.Result["errors"] = errors
	t.task.Result["sent_targets"] = sentTargets
	t.task.Result["target_results"] = targetResults // 添加每个目标的详细结果
	t.task.Result["total_targets"] = len(targets)
	t.task.Result["success_rate"] = float64(sentCount) / float64(len(targets))
	t.task.Result["send_time"] = time.Now().Unix()

	addLog(fmt.Sprintf("任务执行完成: 成功 %d, 失败 %d", sentCount, failedCount))

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

	// 获取自动加群配置
	autoJoin := false
	if val, ok := config["auto_join"].(bool); ok {
		autoJoin = val
	}

	// 获取单号限制
	limitPerAccount := 0
	if val, ok := config["limit_per_account"].(float64); ok {
		limitPerAccount = int(val)
	} else if val, ok := config["limit_per_account"].(int); ok {
		limitPerAccount = int(val)
	}
	// 计算当前账号需要发送的群组范围
	var targetGroups []interface{}

	// 使用 task.Result 中的 next_group_index 来追踪进度
	startIndex := 0
	if val, ok := t.task.Result["next_group_index"].(float64); ok {
		startIndex = int(val)
	}

	if limitPerAccount > 0 {
		endIndex := startIndex + limitPerAccount
		if endIndex > len(groups) {
			endIndex = len(groups)
		}

		if startIndex < len(groups) {
			targetGroups = groups[startIndex:endIndex]
			// 更新进度
			t.task.Result["next_group_index"] = float64(endIndex)
		} else {
			targetGroups = []interface{}{}
		}
	} else {
		// 如果没有限制，发送给所有群组
		targetGroups = groups
	}

	// 记录本次执行的范围，便于调试
	t.task.Result[fmt.Sprintf("account_range_%d", time.Now().UnixNano())] = fmt.Sprintf("%d-%d", startIndex, startIndex+len(targetGroups))

	// 获取发送间隔 (防止被限制)
	intervalSec := 3 // 默认3秒间隔，群发更谨慎
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	// 初始化日志
	var logs []string
	if existingLogs, ok := t.task.Result["logs"].([]interface{}); ok {
		for _, l := range existingLogs {
			if str, ok := l.(string); ok {
				logs = append(logs, str)
			}
		}
	}

	addLog := func(msg string) {
		logEntry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		logs = append(logs, logEntry)
		t.task.Result["logs"] = logs
	}

	addLog(fmt.Sprintf("开始执行群发任务，目标群组数: %d", len(targetGroups)))

	sentCount := 0
	failedCount := 0
	var errors []string
	var sentGroups []string

	// 发送消息到每个群组
	for i, group := range targetGroups {
		// 添加发送间隔（除了第一个消息）
		if i > 0 && intervalSec > 0 {
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}

		var explicitPeer tg.InputPeerClass
		var joinErr error

		// 如果开启了自动加群，尝试先加入
		if autoJoin {
			addLog(fmt.Sprintf("尝试自动加入群组: %v", group))
			explicitPeer, joinErr = t.joinGroup(ctx, api, group)
			if joinErr != nil {
				// 记录加群失败，但仍尝试发送（可能已经在群里了）
				addLog(fmt.Sprintf("自动加群失败: %v, 尝试直接发送", joinErr))
			} else {
				addLog(fmt.Sprintf("自动加群成功: %v", group))
				// 加群成功后稍微等待一下，确保状态同步
				time.Sleep(1 * time.Second)
			}
		}

		err := t.sendBroadcastMessage(ctx, api, group, message, explicitPeer)
		if err != nil {
			errMsg := fmt.Sprintf("发送失败 [%v]: %v", group, err)
			addLog(errMsg)
			errors = append(errors, errMsg)
			failedCount++
		} else {
			addLog(fmt.Sprintf("发送成功: %v", group))
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
	t.task.Result["logs"] = logs
	t.task.Result["sent_groups"] = sentGroups
	t.task.Result["total_groups"] = len(targetGroups)
	if len(targetGroups) > 0 {
		t.task.Result["success_rate"] = float64(sentCount) / float64(len(targetGroups))
	} else {
		t.task.Result["success_rate"] = 0
	}
	t.task.Result["send_time"] = time.Now().Unix()

	addLog(fmt.Sprintf("任务执行完成: 成功 %d, 失败 %d", sentCount, failedCount))

	return nil
}

// joinGroup 尝试加入群组，并返回 InputPeer
func (t *BroadcastTask) joinGroup(ctx context.Context, api *tg.Client, group interface{}) (tg.InputPeerClass, error) {
	groupStr, ok := group.(string)
	if !ok {
		return nil, nil // 非字符串无法通过此方法加入
	}

	// 处理链接或用户名
	cleanGroupname := groupStr
	if len(groupStr) > 0 && groupStr[0] == '@' {
		cleanGroupname = groupStr[1:]
	}

	// 检查是否是邀请链接 (t.me/joinchat/...)
	if strings.Contains(cleanGroupname, "joinchat/") {
		hash := cleanGroupname[strings.Index(cleanGroupname, "joinchat/")+9:]
		if hash == "" {
			return nil, fmt.Errorf("invalid join link")
		}
		updates, err := api.MessagesImportChatInvite(ctx, hash)
		if err != nil {
			if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
				// 如果已经在群里，我们无法直接获取 InputPeer，因为 CheckChatInvite 不返回 ID
				// 只能返回 nil，让 sendBroadcastMessage 尝试通过其他方式（如 Dialogs）解决
				// 或者这里可以尝试 Search?
				return nil, nil
			}
			return nil, err
		}

		// 从 Updates 中提取 Chat/Channel
		return t.extractInputPeerFromUpdates(updates)
	}

	// 移除其他链接前缀
	if len(cleanGroupname) > 13 && cleanGroupname[:13] == "https://t.me/" {
		cleanGroupname = cleanGroupname[13:]
	} else if len(cleanGroupname) > 5 && cleanGroupname[:5] == "t.me/" {
		cleanGroupname = cleanGroupname[5:]
	}

	// 解析用户名
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: cleanGroupname,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve group: %w", err)
	}

	// 尝试加入
	if len(resolved.Chats) > 0 {
		if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
			inputChannel := &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}

			// 检查是否已经在群里 (Left=false)
			if !channel.Left {
				return &tg.InputPeerChannel{
					ChannelID:  channel.ID,
					AccessHash: channel.AccessHash,
				}, nil
			}

			_, err := api.ChannelsJoinChannel(ctx, inputChannel)
			if err != nil {
				return nil, err
			}

			return &tg.InputPeerChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}, nil
		}
	}

	return nil, fmt.Errorf("group not found or not a channel/supergroup")
}

// extractInputPeerFromUpdates 从 Updates 中提取 InputPeer
func (t *BroadcastTask) extractInputPeerFromUpdates(updates tg.UpdatesClass) (tg.InputPeerClass, error) {
	var chats []tg.ChatClass

	switch v := updates.(type) {
	case *tg.Updates:
		chats = v.Chats
	case *tg.UpdatesCombined:
		chats = v.Chats
	}

	if len(chats) > 0 {
		return t.extractInputPeerFromChat(chats[0])
	}
	return nil, fmt.Errorf("no chats found in updates")
}

// extractInputPeerFromChat 从 ChatClass 提取 InputPeer
func (t *BroadcastTask) extractInputPeerFromChat(chat tg.ChatClass) (tg.InputPeerClass, error) {
	switch c := chat.(type) {
	case *tg.Chat:
		return &tg.InputPeerChat{ChatID: c.ID}, nil
	case *tg.Channel:
		return &tg.InputPeerChannel{
			ChannelID:  c.ID,
			AccessHash: c.AccessHash,
		}, nil
	}
	return nil, fmt.Errorf("unknown chat type")
}

// sendBroadcastMessage 发送群发消息到指定群组
func (t *BroadcastTask) sendBroadcastMessage(ctx context.Context, api *tg.Client, group interface{}, message string, explicitPeer tg.InputPeerClass) error {
	var inputPeer tg.InputPeerClass

	// 如果提供了明确的 Peer (通常来自 joinGroup)，直接使用
	if explicitPeer != nil {
		inputPeer = explicitPeer
	} else {
		switch v := group.(type) {
		case int64:
			inputPeer = &tg.InputPeerChat{ChatID: v}
		case float64:
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

			// 移除 joinchat 前缀
			if strings.Contains(cleanGroupname, "joinchat/") {
				return fmt.Errorf("cannot send message to invite link directly, please ensure auto_join is enabled and successful")
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

	// 初始化日志
	var logs []string
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	addLog := func(msg string) {
		logEntry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		logs = append(logs, logEntry)
		t.task.Result["logs"] = logs
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

	addLog(fmt.Sprintf("开始监听验证码，超时时间: %d秒", timeoutSec))
	addLog(fmt.Sprintf("监听发送者: %v", senders))

	startTime := time.Now()
	var verifyCode string
	var receivedAt time.Time
	var senderInfo string

	// 轮询检查新消息
	lastLogTime := time.Now()
	for time.Since(startTime) < time.Duration(timeoutSec)*time.Second {
		// 每30秒打印一次心跳日志
		if time.Since(lastLogTime) > 30*time.Second {
			addLog(fmt.Sprintf("正在监听中... (已等待 %d 秒)", int(time.Since(startTime).Seconds())))
			lastLogTime = time.Now()
		}

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
			addLog(fmt.Sprintf("成功接收到验证码: %s (来自: %s)", code, sender))
			break
		}

		// 等待2秒后再次检查
		time.Sleep(2 * time.Second)
	}

	// 更新任务结果
	if verifyCode != "" {
		t.task.Result["verify_code"] = verifyCode
		t.task.Result["sender"] = senderInfo
		t.task.Result["received_at"] = receivedAt.Unix()
		t.task.Result["status"] = "received"
	} else {
		t.task.Result["verify_code"] = ""
		t.task.Result["status"] = "timeout"
		t.task.Result["error"] = "verification code not received within timeout"
		addLog("监听超时，未收到验证码")
	}

	t.task.Result["timeout_seconds"] = timeoutSec
	t.task.Result["actual_wait_seconds"] = int(time.Since(startTime).Seconds())

	return nil
}

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

	// 初始化日志
	var logs []string
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	addLog := func(msg string) {
		logEntry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
		logs = append(logs, logEntry)
		t.task.Result["logs"] = logs
	}

	addLog("开始执行 AI 炒群任务...")

	// 获取目标群组（支持ID和用户名）
	var inputPeer tg.InputPeerClass
	var targetGroupName string

	if groupID, ok := config["group_id"].(float64); ok && groupID > 0 {
		inputPeer = &tg.InputPeerChat{ChatID: int64(groupID)}
		targetGroupName = fmt.Sprintf("ID: %d", int64(groupID))
	} else if groupName, ok := config["group_name"].(string); ok && groupName != "" {
		targetGroupName = groupName
		// 解析群组用户名
		resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: groupName,
		})
		if err != nil {
			addLog(fmt.Sprintf("无法解析群组 %s: %v", groupName, err))
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

	addLog(fmt.Sprintf("目标群组: %s", targetGroupName))

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

	if personality, ok := aiConfig["personality"].(string); ok {
		addLog(fmt.Sprintf("AI 人格: %s", personality))
	}

	// 获取监控时长
	monitorDuration := 300 // 默认5分钟
	if duration, exists := config["monitor_duration_seconds"]; exists {
		if durationFloat, ok := duration.(float64); ok {
			monitorDuration = int(durationFloat)
		}
	}

	addLog(fmt.Sprintf("任务持续时间: %d 秒", monitorDuration))

	responseSent := 0
	messagesProcessed := 0

	// 获取群组最新消息作为初始检查
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		Limit: 5,
	})
	if err != nil {
		addLog(fmt.Sprintf("获取历史消息失败: %v", err))
		return fmt.Errorf("failed to get chat history: %w", err)
	}

	// 分析群聊上下文并可能发送回复
	if messages, ok := history.(*tg.MessagesMessages); ok {
		addLog(fmt.Sprintf("获取到 %d 条历史消息，正在分析...", len(messages.Messages)))
		for _, msg := range messages.Messages {
			if message, ok := msg.(*tg.Message); ok {
				messagesProcessed++

				// 简单的回复逻辑 - 如果消息包含关键词且随机数允许
				if t.shouldRespondSimple(message, aiConfig) {
					response := t.generateSimpleAIResponse(message, aiConfig)
					if response != "" {
						addLog(fmt.Sprintf("触发回复规则 (原文: %s...)", t.truncateString(message.Message, 20)))
						_, err = api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
							Peer:     inputPeer,
							Message:  response,
							RandomID: time.Now().UnixNano(),
						})
						if err == nil {
							responseSent++
							addLog(fmt.Sprintf("发送回复成功: %s", response))
						} else {
							addLog(fmt.Sprintf("发送回复失败: %v", err))
						}
						break // 只发送一个回复
					}
				}
			}
		}
	}

	if responseSent == 0 {
		addLog("本次检查未触发回复")
	}

	// 更新任务结果
	t.task.Result["messages_processed"] = messagesProcessed
	t.task.Result["responses_sent"] = responseSent
	t.task.Result["monitor_duration"] = monitorDuration
	t.task.Result["completion_time"] = time.Now().Unix()

	addLog(fmt.Sprintf("任务完成，处理消息: %d, 发送回复: %d", messagesProcessed, responseSent))

	return nil
}

func (t *GroupChatTask) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
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
