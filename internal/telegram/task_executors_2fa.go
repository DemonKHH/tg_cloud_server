package telegram

import (
	"context"
	"fmt"
	"time"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// Update2FATask 修改2FA密码任务
type Update2FATask struct {
	task *models.Task
}

// NewUpdate2FATask 创建修改2FA密码任务
func NewUpdate2FATask(task *models.Task) *Update2FATask {
	return &Update2FATask{task: task}
}

// Execute 执行修改2FA密码
func (t *Update2FATask) Execute(ctx context.Context, api *tg.Client) error {
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

	addLog("开始执行修改 2FA 密码任务...")

	// 1. 获取配置
	config := t.task.Config
	newPassword, _ := config["new_password"].(string)
	oldPassword, _ := config["old_password"].(string)
	hint, _ := config["hint"].(string)

	// 2. 获取当前密码设置
	addLog("正在获取当前密码设置...")
	// AccountGetPasswordSettings 获取当前密码信息
	passwordSettings, err := api.AccountGetPassword(ctx)
	if err != nil {
		addLog(fmt.Sprintf("获取密码设置失败: %v", err))
		return fmt.Errorf("failed to get password settings: %w", err)
	}

	if passwordSettings.HasPassword {
		addLog("当前账号已设置 2FA 密码，正在验证旧密码...")
		if oldPassword == "" {
			addLog("错误: 未提供旧密码")
			return fmt.Errorf("old_password is required when 2FA is enabled")
		}

		// 验证旧密码
		inputCheck, err := auth.PasswordHash(
			[]byte(oldPassword),
			passwordSettings.SRPID,
			passwordSettings.SRPB,
			passwordSettings.SecureRandom,
			passwordSettings.CurrentAlgo,
		)
		if err != nil {
			addLog(fmt.Sprintf("计算密码哈希失败: %v", err))
			return fmt.Errorf("failed to compute password hash: %w", err)
		}

		// 验证密码
		_, err = api.AuthCheckPassword(ctx, inputCheck)
		if err != nil {
			addLog(fmt.Sprintf("旧密码验证失败: %v", err))
			return fmt.Errorf("invalid old password: %w", err)
		}
		addLog("旧密码验证成功")
	} else {
		addLog("当前账号未设置 2FA 密码")
	}

	// 3. 修改密码
	if newPassword != "" {
		addLog("正在设置新密码...")
		// TODO: 实现 SRP 算法生成新密码的 hash 和 salt
		// 需要实现 PasswordKdfAlgoSHA256SHA256PBKDF2HMACSHA512iter100000SHA256ModPow

		_ = hint // prevent unused error

		addLog("错误: 设置新密码功能暂未实现 (需要 SRP 算法支持)")
		return fmt.Errorf("setting new password is not yet implemented")
	} else {
		addLog("未提供新密码，任务结束")
	}

	return nil
}

// GetType 获取任务类型
func (t *Update2FATask) GetType() string {
	return "update_2fa"
}
