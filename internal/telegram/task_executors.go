package telegram

import (
	"context"
	"fmt"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/tg"
)

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
	// 获取用户信息
	user, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// 检查账号状态 - 使用正确的gotd/td API
	// TODO: 根据实际的gotd/td API结构调整字段访问
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	// 暂时使用简化的结果，需要根据实际API调整
	t.task.Result["user_info"] = "retrieved"
	t.task.Result["status"] = "active"
	// t.task.Result["user_id"] = user.FullUser.GetID()  // 需要验证正确的字段
	// t.task.Result["phone"] = user.FullUser.GetPhone()
	// t.task.Result["username"] = user.FullUser.GetUsername()

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

	// 获取目标用户列表
	targets, ok := config["targets"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid targets configuration")
	}

	// 获取消息内容
	message, ok := config["message"].(string)
	if !ok {
		return fmt.Errorf("invalid message configuration")
	}

	sentCount := 0
	var errors []string

	// 发送私信给每个目标用户
	for _, target := range targets {
		username, ok := target.(string)
		if !ok {
			continue
		}

		// 解析用户名 - 使用正确的Peer类型
		inputPeer := &tg.InputPeerUser{
			UserID: 0, // 需要先解析用户名获取ID
			// AccessHash: 0, // 通常需要access hash
		}

		// 发送消息
		_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:    inputPeer,
			Message: message,
		})

		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to send to %s: %v", username, err))
		} else {
			sentCount++
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["sent_count"] = sentCount
	t.task.Result["total_targets"] = len(targets)
	t.task.Result["errors"] = errors

	return nil
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

	// 获取目标群组列表
	groups, ok := config["groups"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid groups configuration")
	}

	// 获取消息内容
	message, ok := config["message"].(string)
	if !ok {
		return fmt.Errorf("invalid message configuration")
	}

	sentCount := 0
	var errors []string

	// 发送消息到每个群组
	for _, group := range groups {
		chatID, ok := group.(int64)
		if !ok {
			continue
		}

		// 发送消息
		_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer: &tg.InputPeerChat{
				ChatID: chatID,
			},
			Message: message,
		})

		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to send to group %d: %v", chatID, err))
		} else {
			sentCount++
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["sent_count"] = sentCount
	t.task.Result["total_groups"] = len(groups)
	t.task.Result["errors"] = errors

	return nil
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
	// 这是一个监听任务，需要持续监听新消息
	// 实际实现中应该使用Update机制

	// 获取最新消息
	_, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit: 10,
	})
	if err != nil {
		return fmt.Errorf("failed to get dialogs: %w", err)
	}

	// 查找验证码消息
	var verifyCode string
	// 这里应该实现验证码解析逻辑

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["verify_code"] = verifyCode
	t.task.Result["received_at"] = "2023-10-01T10:00:00Z"

	return nil
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

	// 获取目标群组
	groupID, ok := config["group_id"].(int64)
	if !ok {
		return fmt.Errorf("invalid group_id configuration")
	}

	// 获取AI配置
	aiConfig, ok := config["ai_config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid ai_config configuration")
	}

	// 获取群组最新消息
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChat{
			ChatID: groupID,
		},
		Limit: 10,
	})
	if err != nil {
		return fmt.Errorf("failed to get chat history: %w", err)
	}

	// 分析群聊上下文（这里应该调用AI服务）
	// 进行类型断言
	var response string
	if messages, ok := history.(*tg.MessagesMessages); ok {
		response = t.generateAIResponse(messages, aiConfig)
	} else {
		// 处理其他消息类型或使用默认回复
		response = "AI服务暂时不可用"
	}

	// 发送AI生成的回复
	if response != "" {
		_, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer: &tg.InputPeerChat{
				ChatID: groupID,
			},
			Message: response,
		})
		if err != nil {
			return fmt.Errorf("failed to send AI response: %w", err)
		}
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["group_id"] = groupID
	t.task.Result["ai_response"] = response
	t.task.Result["interaction_count"] = 1

	return nil
}

// generateAIResponse 生成AI回复（实际应该调用AI服务）
func (t *GroupChatTask) generateAIResponse(history *tg.MessagesMessages, config map[string]interface{}) string {
	// 这里应该调用AI服务生成智能回复
	// 临时返回示例回复
	return "这是一个AI生成的回复"
}

// GetType 获取任务类型
func (t *GroupChatTask) GetType() string {
	return "group_chat"
}
