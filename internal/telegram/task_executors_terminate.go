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

	addLog("开始获取当前活动会话列表...")

	// 1. 获取当前所有授权
	authorizations, err := api.AccountGetAuthorizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authorizations: %w", err)
	}

	totalSessions := len(authorizations.Authorizations)
	addLog(fmt.Sprintf("获取成功，当前共有 %d 个活动会话", totalSessions))

	// 记录详细的会话信息
	terminatedCount := 0
	for _, auth := range authorizations.Authorizations {
		if auth.Current {
			addLog(fmt.Sprintf("保留当前会话: %s (%s) - IP: %s", auth.DeviceModel, auth.Platform, auth.IP))
			continue
		}

		addLog(fmt.Sprintf("准备踢出设备: %s (%s) - IP: %s, 登录时间: %s",
			auth.DeviceModel,
			auth.Platform,
			auth.IP,
			time.Unix(int64(auth.DateCreated), 0).Format("2006-01-02 15:04:05"),
		))
		terminatedCount++
	}

	if terminatedCount == 0 {
		addLog("没有发现其他设备，无需踢出")
		t.task.Result["terminated_count"] = 0
		t.task.Result["status"] = "success"
		t.task.Result["executed_at"] = time.Now().Unix()
		return nil
	}

	addLog(fmt.Sprintf("正在执行踢出操作，将踢出 %d 个设备...", terminatedCount))

	// 2. 踢出其他设备
	// ResetAuthorizations 会踢出除当前会话外的所有其他会话
	success, err := api.AuthResetAuthorizations(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("踢出操作失败: %v", err)
		addLog(errMsg)
		return fmt.Errorf("failed to reset authorizations: %w", err)
	}

	if !success {
		errMsg := "踢出操作返回失败 (false)"
		addLog(errMsg)
		return fmt.Errorf("failed to reset authorizations (returned false)")
	}

	addLog("踢出操作执行成功！")

	// 更新任务结果
	t.task.Result["terminated_count"] = terminatedCount
	t.task.Result["status"] = "success"
	t.task.Result["executed_at"] = time.Now().Unix()

	return nil
}

// GetType 获取任务类型
func (t *TerminateSessionsTask) GetType() string {
	return "terminate_sessions"
}
