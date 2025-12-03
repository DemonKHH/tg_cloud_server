package telegram

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"math/big"
	"time"

	"tg_cloud_server/internal/models"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/crypto/pbkdf2"
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

	var currentPassword tg.InputCheckPasswordSRPClass
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
		currentPassword = inputCheck
	} else {
		addLog("当前账号未设置 2FA 密码")
		currentPassword = &tg.InputCheckPasswordEmpty{}
	}

	// 3. 修改密码
	if newPassword != "" {
		addLog("正在设置新密码...")

		// 生成随机 salt
		salt := make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}

		// RFC 5054 2048-bit Prime (Group 14)
		pHex := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF"

		p := new(big.Int)
		p.SetString(pHex, 16)
		g := 2 // Default generator

		// Implementation of PasswordKdfAlgoSHA256SHA256PBKDF2HMACSHA512iter100000SHA256ModPow
		// 1. SHA256(password)
		// 2. SHA256(prev)
		// 3. PBKDF2(prev, salt, 100000, 64, SHA512)
		// 4. ModPow: g^x mod p (where x is from PBKDF2)

		h1 := sha256.Sum256([]byte(newPassword))
		h2 := sha256.Sum256(h1[:])

		pbkdf2Out := pbkdf2.Key(h2[:], salt, 100000, 64, sha512.New)

		x := new(big.Int).SetBytes(pbkdf2Out)
		gBig := big.NewInt(int64(g))
		v := new(big.Int).Exp(gBig, x, p)

		vBytes := v.Bytes()
		// Pad to 256 bytes if necessary (2048 bits)
		if len(vBytes) < 256 {
			padded := make([]byte, 256)
			copy(padded[256-len(vBytes):], vBytes)
			vBytes = padded
		}

		newAlgo := &tg.PasswordKdfAlgoSHA256SHA256PBKDF2HMACSHA512iter100000SHA256ModPow{
			Salt1: salt,
			Salt2: []byte{}, // Usually empty
			G:     g,
			P:     p.Bytes(),
		}

		settings := &tg.AccountPasswordInputSettings{
			NewAlgo:         newAlgo,
			NewPasswordHash: vBytes,
			Hint:            hint,
		}

		req := &tg.AccountUpdatePasswordSettingsRequest{
			Password:    currentPassword,
			NewSettings: *settings,
		}

		if _, err := api.AccountUpdatePasswordSettings(ctx, req); err != nil {
			addLog(fmt.Sprintf("设置新密码失败: %v", err))
			return fmt.Errorf("failed to update password settings: %w", err)
		}

		addLog("新密码设置成功")
	} else {
		addLog("未提供新密码，任务结束")
	}

	return nil
}

// GetType 获取任务类型
func (t *Update2FATask) GetType() string {
	return "update_2fa"
}
