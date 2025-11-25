package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/tg"
)

// JoinGroupTask 批量加群任务
type JoinGroupTask struct {
	task *models.Task
}

// NewJoinGroupTask 创建批量加群任务
func NewJoinGroupTask(task *models.Task) *JoinGroupTask {
	return &JoinGroupTask{task: task}
}

// Execute 执行批量加群
func (t *JoinGroupTask) Execute(ctx context.Context, api *tg.Client) error {
	config := t.task.Config

	// 验证配置完整性
	if config == nil {
		return fmt.Errorf("task config is nil")
	}

	// 获取目标群组列表
	groups, ok := config["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		return fmt.Errorf("invalid or empty groups configuration")
	}

	// 获取间隔时间
	intervalSec := 5 // 默认5秒间隔
	if interval, exists := config["interval_seconds"]; exists {
		if intervalFloat, ok := interval.(float64); ok {
			intervalSec = int(intervalFloat)
		}
	}

	successCount := 0
	failedCount := 0
	var errors []string
	var joinedGroups []string
	groupResults := make(map[string]interface{})

	// 遍历群组进行加入
	for i, group := range groups {
		// 添加间隔（除了第一个）
		if i > 0 && intervalSec > 0 {
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}

		groupStr, ok := group.(string)
		if !ok {
			errorMsg := fmt.Sprintf("invalid group format: %v", group)
			errors = append(errors, errorMsg)
			failedCount++
			continue
		}

		// 记录开始时间
		startTime := time.Now()

		// 执行加入逻辑
		err := t.joinGroup(ctx, api, groupStr)
		duration := time.Since(startTime)

		if err != nil {
			errorMsg := fmt.Sprintf("failed to join %s: %v", groupStr, err)
			errors = append(errors, errorMsg)
			groupResults[groupStr] = map[string]interface{}{
				"status":   "failed",
				"error":    err.Error(),
				"duration": duration.String(),
			}
			failedCount++
		} else {
			successCount++
			joinedGroups = append(joinedGroups, groupStr)
			groupResults[groupStr] = map[string]interface{}{
				"status":   "success",
				"duration": duration.String(),
			}
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["joined_count"] = successCount
	t.task.Result["failed_count"] = failedCount
	t.task.Result["errors"] = errors
	t.task.Result["joined_groups"] = joinedGroups
	t.task.Result["group_results"] = groupResults
	t.task.Result["total_groups"] = len(groups)
	t.task.Result["success_rate"] = float64(successCount) / float64(len(groups))
	t.task.Result["completion_time"] = time.Now().Unix()

	return nil
}

// joinGroup 加入单个群组
func (t *JoinGroupTask) joinGroup(ctx context.Context, api *tg.Client, groupInput string) error {
	// 1. 处理 Invite Link (t.me/+hash 或 t.me/joinchat/hash)
	if t.isInviteLink(groupInput) {
		hash := t.extractInviteHash(groupInput)
		if hash == "" {
			return fmt.Errorf("invalid invite link format")
		}

		_, err := api.MessagesImportChatInvite(ctx, hash)
		return err
	}

	// 2. 处理公开用户名/链接
	username := t.extractUsername(groupInput)
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
				return err
			}
			// 已经是成员，视为成功
			return nil
		}
		// 普通群组通常不能通过 resolve username 直接加入，除非被邀请，
		// 但如果 resolve 成功，它通常是公开群，应该作为 channel 处理 (supergroup is a channel in API)
		// 如果是 Chat 类型，通常意味着它是 basic group，且你已经在里面了或者它是通过其他方式获取的。
		// 公开群在 API 中基本都是 Channel (Supergroup)。
		return fmt.Errorf("target is not a channel or supergroup")
	}

	return fmt.Errorf("group not found")
}

// isInviteLink 检查是否为邀请链接
func (t *JoinGroupTask) isInviteLink(input string) bool {
	return t.contains(input, "joinchat") || t.contains(input, "/+")
}

// extractInviteHash 提取邀请哈希
func (t *JoinGroupTask) extractInviteHash(input string) string {
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
func (t *JoinGroupTask) extractUsername(input string) string {
	// 移除 https://t.me/ 或 t.me/
	input = strings.TrimPrefix(input, "https://")
	input = strings.TrimPrefix(input, "http://")
	input = strings.TrimPrefix(input, "t.me/")
	input = strings.TrimPrefix(input, "@")

	// 移除可能的参数
	if idx := strings.Index(input, "?"); idx != -1 {
		input = input[:idx]
	}
	if idx := strings.Index(input, "/"); idx != -1 {
		input = input[:idx]
	}

	return input
}

// contains 检查字符串包含 (复用 VerifyCodeTask 的逻辑，或者重新实现)
func (t *JoinGroupTask) contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// GetType 获取任务类型
func (t *JoinGroupTask) GetType() string {
	return "join_group"
}
