package telegram

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/tg"
)

// ForceAddGroupTask 强拉进群任务
type ForceAddGroupTask struct {
	task      *models.Task
	accountID uint64
}

// NewForceAddGroupTask 创建强拉进群任务
func NewForceAddGroupTask(task *models.Task, accountID uint64) *ForceAddGroupTask {
	return &ForceAddGroupTask{
		task:      task,
		accountID: accountID,
	}
}

// Execute 执行强拉进群
func (t *ForceAddGroupTask) Execute(ctx context.Context, api *tg.Client) error {
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
	allTargets, ok := config["targets"].([]interface{})
	if !ok || len(allTargets) == 0 {
		return fmt.Errorf("invalid or empty targets configuration")
	}

	// 获取间隔时间
	intervalSec := 5 // 默认5秒间隔
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	// 获取单号限制
	limitPerAccount := 0
	if limit, exists := config["limit_per_account"]; exists {
		if limitFloat, ok := limit.(float64); ok {
			limitPerAccount = int(limitFloat)
		}
	}

	// 确定当前账号需要处理的目标列表
	var myTargets []interface{}
	if limitPerAccount > 0 {
		// 获取所有账号ID列表
		accountIDs := t.task.GetAccountIDList()

		// 找到当前账号的索引
		myIndex := -1
		for i, id := range accountIDs {
			if id == t.accountID {
				myIndex = i
				break
			}
		}

		if myIndex == -1 {
			return fmt.Errorf("current account ID %d not found in task account list", t.accountID)
		}

		// 计算切片范围
		start := myIndex * limitPerAccount
		end := start + limitPerAccount

		// 检查是否超出范围
		if start >= len(allTargets) {
			// 该账号没有分配到任务
			t.updateResult(0, 0, nil, nil, nil)
			addLog("该账号未分配到任务目标 (超出范围)")
			return nil
		}

		// 截取目标
		if end > len(allTargets) {
			end = len(allTargets)
		}
		myTargets = allTargets[start:end]
	} else {
		// 如果没有限制，默认所有账号处理所有目标（这通常不是预期的，但在没有明确分片逻辑时可能是为了冗余）
		// 或者更合理的逻辑是：如果没有 limit_per_account，则平均分配？
		// 根据用户需求，这里假设没有 limit_per_account 时，所有账号尝试拉取所有目标（可能会有重复尝试错误，但能保证覆盖）
		// 为了安全起见，如果没有 limit_per_account，我们还是建议平均分配，或者让用户明确指定。
		// 这里暂且按照“所有目标”处理，或者我们可以实现一个简单的平均分配逻辑作为默认值。
		// 鉴于强拉的高风险，默认平均分配比较好。

		accountIDs := t.task.GetAccountIDList()
		totalAccounts := len(accountIDs)
		if totalAccounts > 0 {
			myIndex := -1
			for i, id := range accountIDs {
				if id == t.accountID {
					myIndex = i
					break
				}
			}

			if myIndex != -1 {
				// 平均分配
				perAccount := int(math.Ceil(float64(len(allTargets)) / float64(totalAccounts)))
				start := myIndex * perAccount
				end := start + perAccount
				if start < len(allTargets) {
					if end > len(allTargets) {
						end = len(allTargets)
					}
					myTargets = allTargets[start:end]
				}
			}
		}

		// 如果上述逻辑未执行（例如找不到索引），则回退到空列表
		if myTargets == nil {
			myTargets = []interface{}{}
		}
	}

	if len(myTargets) == 0 {
		t.updateResult(0, 0, nil, nil, nil)
		addLog("该账号未分配到任务目标")
		return nil
	}

	addLog(fmt.Sprintf("开始执行强拉任务，目标数: %d，间隔: %d秒", len(myTargets), intervalSec))

	// 解析目标群组
	var inputPeer tg.InputPeerClass
	var isChannel bool // 区分 Channel/Supergroup 和 Basic Group
	var targetGroupName string

	// 自动加群逻辑
	autoJoin := false
	if val, ok := config["auto_join"].(bool); ok {
		autoJoin = val
	}

	if autoJoin {
		var groupToJoin interface{}
		if name, ok := config["group_name"].(string); ok && name != "" {
			groupToJoin = name
		}

		if groupToJoin != nil {
			addLog(fmt.Sprintf("正在尝试自动加群: %v", groupToJoin))
			peer, err := t.joinGroup(ctx, api, groupToJoin)
			if err != nil {
				// 即使加群失败（可能是已经在群里，或者其他原因），也尝试继续解析
				addLog(fmt.Sprintf("自动加群尝试结束: %v", err))
			} else if peer != nil {
				inputPeer = peer
				targetGroupName = fmt.Sprintf("%v", groupToJoin)

				// 判断是否为频道/超级群
				switch peer.(type) {
				case *tg.InputPeerChannel:
					isChannel = true
				default:
					isChannel = false
				}
				addLog("自动加群成功或已获取群组信息")
			}
		}
	}

	if inputPeer == nil {

		if groupName, ok := config["group_name"].(string); ok && groupName != "" {
			targetGroupName = groupName
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
					isChannel = false
				} else if channel, ok := resolved.Chats[0].(*tg.Channel); ok {
					inputPeer = &tg.InputPeerChannel{
						ChannelID:  channel.ID,
						AccessHash: channel.AccessHash,
					}
					isChannel = true
				}
			}
		} else {
			return fmt.Errorf("missing group_name configuration")
		}
	}

	if inputPeer == nil {
		return fmt.Errorf("failed to resolve target group")
	}

	addLog(fmt.Sprintf("目标群组: %s", targetGroupName))

	successCount := 0
	failedCount := 0
	var errors []string
	var addedTargets []string
	targetResults := make(map[string]interface{})

	// 遍历目标进行拉取
	for i, target := range myTargets {
		// 添加间隔（除了第一个）
		if i > 0 && intervalSec > 0 {
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}

		targetStr, ok := target.(string)
		if !ok {
			continue
		}

		startTime := time.Now()
		var err error

		// 解析目标用户
		var userInput tg.InputUserClass
		// 尝试解析用户名
		cleanTarget := strings.TrimPrefix(targetStr, "@")
		resolvedUser, resolveErr := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: cleanTarget,
		})

		if resolveErr != nil {
			err = fmt.Errorf("resolve user failed: %w", resolveErr)
		} else if len(resolvedUser.Users) > 0 {
			if user, ok := resolvedUser.Users[0].(*tg.User); ok {
				userInput = &tg.InputUser{
					UserID:     user.ID,
					AccessHash: user.AccessHash,
				}
			} else {
				err = fmt.Errorf("resolved peer is not a user")
			}
		} else {
			err = fmt.Errorf("user not found")
		}

		if err == nil {
			// 执行拉人
			if isChannel {
				// 频道/超级群
				channelPeer := inputPeer.(*tg.InputPeerChannel)
				_, err = api.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
					Channel: &tg.InputChannel{
						ChannelID:  channelPeer.ChannelID,
						AccessHash: channelPeer.AccessHash,
					},
					Users: []tg.InputUserClass{userInput},
				})
			} else {
				// 普通群
				_, err = api.MessagesAddChatUser(ctx, &tg.MessagesAddChatUserRequest{
					ChatID:   inputPeer.(*tg.InputPeerChat).ChatID, // 安全断言
					UserID:   userInput,
					FwdLimit: 0, // 默认显示历史消息数为0
				})
			}
		}

		duration := time.Since(startTime)

		if err != nil {
			// 忽略一些常见的非致命错误，例如用户已经在群里
			if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
				// 视为成功或跳过
				successCount++
				addedTargets = append(addedTargets, targetStr)
				targetResults[targetStr] = map[string]interface{}{
					"status":   "success", // 或者 "already_in"
					"note":     "already participant",
					"duration": duration.String(),
				}
				addLog(fmt.Sprintf("用户已在群中: %s", targetStr))
			} else {
				errorMsg := fmt.Sprintf("failed to add %s: %v", targetStr, err)
				errors = append(errors, errorMsg)
				targetResults[targetStr] = map[string]interface{}{
					"status":   "failed",
					"error":    err.Error(),
					"duration": duration.String(),
				}
				failedCount++
				addLog(fmt.Sprintf("拉人失败 [%s]: %v", targetStr, err))
			}
		} else {
			successCount++
			addedTargets = append(addedTargets, targetStr)
			targetResults[targetStr] = map[string]interface{}{
				"status":   "success",
				"duration": duration.String(),
			}
			addLog(fmt.Sprintf("拉人成功: %s", targetStr))
		}
	}

	t.updateResult(successCount, failedCount, errors, addedTargets, targetResults)
	addLog(fmt.Sprintf("任务执行完成: 成功 %d, 失败 %d", successCount, failedCount))

	return nil
}

// updateResult 更新任务结果
func (t *ForceAddGroupTask) updateResult(success, failed int, errors []string, added []string, details map[string]interface{}) {
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["added_count"] = success
	t.task.Result["failed_count"] = failed
	if len(errors) > 0 {
		t.task.Result["errors"] = errors
	}
	if len(added) > 0 {
		t.task.Result["added_targets"] = added
	}
	if len(details) > 0 {
		t.task.Result["target_results"] = details
	}
	t.task.Result["completion_time"] = time.Now().Unix()
}

// GetType 获取任务类型
func (t *ForceAddGroupTask) GetType() string {
	return "force_add_group"
}

// joinGroup 尝试加入群组，并返回 InputPeer
func (t *ForceAddGroupTask) joinGroup(ctx context.Context, api *tg.Client, group interface{}) (tg.InputPeerClass, error) {
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
				// 只能返回 nil，让外部尝试通过其他方式（如 Dialogs）解决
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
func (t *ForceAddGroupTask) extractInputPeerFromUpdates(updates tg.UpdatesClass) (tg.InputPeerClass, error) {
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
func (t *ForceAddGroupTask) extractInputPeerFromChat(chat tg.ChatClass) (tg.InputPeerClass, error) {
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
