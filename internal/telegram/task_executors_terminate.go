package telegram

import (
	"context"
	"fmt"
	"time"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/tg"
)

// TerminateSessionsTask 踢出其他设备任务
type TerminateSessionsTask struct {
	task *models.Task
}

// NewTerminateSessionsTask 创建踢出其他设备任务
func NewTerminateSessionsTask(task *models.Task) *TerminateSessionsTask {
	return &TerminateSessionsTask{task: task}
}

// Execute 执行踢出其他设备
func (t *TerminateSessionsTask) Execute(ctx context.Context, api *tg.Client) error {
	// 1. 获取当前所有授权
	authorizations, err := api.AccountGetAuthorizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authorizations: %w", err)
	}

	// 2. 踢出其他设备
	// ResetAuthorizations 会踢出除当前会话外的所有其他会话
	success, err := api.AuthResetAuthorizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset authorizations: %w", err)
	}

	if !success {
		return fmt.Errorf("failed to reset authorizations (returned false)")
	}

	// 3. 统计踢出的设备数量
	// 注意：ResetAuthorizations 成功后，authorizations 列表中的非当前会话都被踢出了
	// 但我们无法确切知道踢出了多少个，只能根据之前的列表估算
	// 实际上，我们只需要知道操作成功即可

	terminatedCount := 0
	if len(authorizations.Authorizations) > 1 {
		terminatedCount = len(authorizations.Authorizations) - 1
	}

	// 更新任务结果
	if t.task.Result == nil {
		t.task.Result = make(models.TaskResult)
	}

	t.task.Result["terminated_count"] = terminatedCount
	t.task.Result["status"] = "success"
	t.task.Result["executed_at"] = time.Now().Unix()

	return nil
}

// GetType 获取任务类型
func (t *TerminateSessionsTask) GetType() string {
	return "terminate_sessions"
}
