package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/telegram"
)

// VerifyCodeService 验证码服务
type VerifyCodeService struct {
	accountRepo    repository.AccountRepository
	userRepo       repository.UserRepository
	connectionPool *telegram.ConnectionPool
	logger         *zap.Logger

	// 临时code存储 (生产环境应使用Redis)
	sessions map[string]*models.VerifyCodeSession
	mutex    sync.RWMutex
}

// NewVerifyCodeService 创建验证码服务
func NewVerifyCodeService(
	accountRepo repository.AccountRepository,
	userRepo repository.UserRepository,
	connectionPool *telegram.ConnectionPool,
	logger *zap.Logger,
) *VerifyCodeService {
	service := &VerifyCodeService{
		accountRepo:    accountRepo,
		userRepo:       userRepo,
		connectionPool: connectionPool,
		logger:         logger.Named("verify_code_service"),
		sessions:       make(map[string]*models.VerifyCodeSession),
	}

	// 启动清理过期会话的协程
	go service.cleanupExpiredSessions()

	return service
}

// GenerateCode 生成临时访问代码
func (s *VerifyCodeService) GenerateCode(userID, accountID uint64, expiresIn int) (*models.GenerateCodeResponse, error) {
	// 验证用户状态
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		s.logger.Warn("User not found",
			zap.Uint64("user_id", userID))
		return nil, models.ErrAccountNotFound
	}

	// 检查用户是否有效（激活且未过期）
	if !user.IsValidUser() {
		if user.IsExpired() {
			s.logger.Warn("User account expired",
				zap.Uint64("user_id", userID),
				zap.Time("expires_at", *user.ExpiresAt))
			return nil, models.NewUserExpiredError(user)
		}
		if !user.IsActive {
			s.logger.Warn("User account inactive",
				zap.Uint64("user_id", userID))
			return nil, fmt.Errorf("user account is disabled")
		}
	}

	// 验证账号权限
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		s.logger.Warn("Account not found or no permission",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID))
		return nil, models.ErrAccountNotFound
	}

	// 设置默认过期时间
	if expiresIn <= 0 {
		expiresIn = 300 // 5分钟
	}
	if expiresIn > 3600 {
		expiresIn = 3600 // 最多1小时
	}

	// 生成唯一代码
	code, err := s.generateUniqueCode()
	if err != nil {
		s.logger.Error("Failed to generate unique code", zap.Error(err))
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// 创建会话
	session := &models.VerifyCodeSession{
		Code:      code,
		AccountID: accountID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	// 存储会话
	s.mutex.Lock()
	s.sessions[code] = session
	s.mutex.Unlock()

	s.logger.Info("Verification code session created",
		zap.String("code", code),
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID),
		zap.String("account_phone", account.Phone),
		zap.Time("expires_at", session.ExpiresAt))

	// 构造响应
	response := &models.GenerateCodeResponse{
		Code:      code,
		URL:       fmt.Sprintf("/api/v1/verify-code/%s", code),
		ExpiresAt: session.ExpiresAt.Unix(),
		ExpiresIn: expiresIn,
	}

	return response, nil
}

// GetVerifyCode 通过code获取验证码
func (s *VerifyCodeService) GetVerifyCode(ctx context.Context, code string, timeoutSeconds int) (*models.VerifyCodeResponse, error) {
	// 获取会话
	s.mutex.RLock()
	session, exists := s.sessions[code]
	s.mutex.RUnlock()

	if !exists {
		return &models.VerifyCodeResponse{
			Success: false,
			Message: models.ErrCodeNotFound.Message,
		}, models.ErrCodeNotFound
	}

	// 检查会话有效性（只检查过期时间）
	if !session.IsValid() {
		return &models.VerifyCodeResponse{
			Success: false,
			Message: models.ErrCodeExpired.Message,
		}, models.ErrCodeExpired
	}

	// 验证用户状态
	user, err := s.userRepo.GetByID(session.UserID)
	if err != nil {
		s.logger.Warn("User not found for code session",
			zap.String("code", code),
			zap.Uint64("user_id", session.UserID))
		return &models.VerifyCodeResponse{
			Success: false,
			Message: models.ErrAccountNotFound.Message,
		}, models.ErrAccountNotFound
	}

	// 检查用户是否有效（激活且未过期）
	if !user.IsValidUser() {
		if user.IsExpired() {
			s.logger.Warn("User account expired during code retrieval",
				zap.String("code", code),
				zap.Uint64("user_id", session.UserID),
				zap.Time("expires_at", *user.ExpiresAt))
			return &models.VerifyCodeResponse{
				Success: false,
				Message: models.NewUserExpiredError(user).Message,
			}, models.NewUserExpiredError(user)
		}
		if !user.IsActive {
			s.logger.Warn("User account inactive during code retrieval",
				zap.String("code", code),
				zap.Uint64("user_id", session.UserID))
			return &models.VerifyCodeResponse{
				Success: false,
				Message: "用户账号已被禁用",
			}, fmt.Errorf("user account is disabled")
		}
	}

	// 获取账号信息
	account, err := s.accountRepo.GetByID(session.AccountID)
	if err != nil {
		s.logger.Error("Failed to get account",
			zap.String("code", code),
			zap.Uint64("account_id", session.AccountID),
			zap.Error(err))
		return &models.VerifyCodeResponse{
			Success: false,
			Message: models.ErrAccountNotFound.Message,
		}, models.ErrAccountNotFound
	}

	s.logger.Info("Starting verification code retrieval",
		zap.String("code", code),
		zap.Uint64("account_id", account.ID),
		zap.String("account_phone", account.Phone),
		zap.Int("timeout_seconds", timeoutSeconds))

	// 设置默认超时时间
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60 // 默认60秒
	}
	if timeoutSeconds > 300 {
		timeoutSeconds = 300 // 最多5分钟
	}

	// 创建验证码获取任务
	task := &verifyCodeTask{
		timeoutSeconds: timeoutSeconds,
		logger:         s.logger,
	}

	// 执行任务获取验证码
	startTime := time.Now()
	accountIDStr := fmt.Sprintf("%d", account.ID)
	err = s.connectionPool.ExecuteTask(accountIDStr, task)
	waitSeconds := int(time.Since(startTime).Seconds())

	if err != nil {
		s.logger.Error("Failed to execute verification code task",
			zap.String("code", code),
			zap.Uint64("account_id", account.ID),
			zap.Error(err))
		return &models.VerifyCodeResponse{
			Success: false,
			Message: models.ErrTelegramConnection.Message,
		}, models.ErrTelegramConnection
	}

	// 获取任务结果
	verifyCodeResult := task.result.code
	senderInfo := task.result.sender
	receivedAt := task.result.receivedAt
	success := task.result.success

	if success {
		s.logger.Info("Verification code received successfully",
			zap.String("code", code),
			zap.Uint64("account_id", account.ID),
			zap.String("verify_code", verifyCodeResult),
			zap.String("sender", senderInfo),
			zap.Int("wait_seconds", waitSeconds))

		return &models.VerifyCodeResponse{
			Success:     true,
			Code:        verifyCodeResult,
			Message:     "验证码获取成功",
			Sender:      senderInfo,
			ReceivedAt:  receivedAt.Unix(),
			WaitSeconds: waitSeconds,
		}, nil
	} else {
		s.logger.Warn("Verification code timeout",
			zap.String("code", code),
			zap.Uint64("account_id", account.ID),
			zap.Int("timeout_seconds", timeoutSeconds),
			zap.Int("wait_seconds", waitSeconds))

		return &models.VerifyCodeResponse{
			Success:     false,
			Message:     fmt.Sprintf("验证码接收超时（等待了%d秒）", waitSeconds),
			WaitSeconds: waitSeconds,
		}, models.ErrVerifyTimeout
	}
}

// generateUniqueCode 生成唯一代码
func (s *VerifyCodeService) generateUniqueCode() (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		// 生成32字节随机数据
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}

		code := hex.EncodeToString(bytes)

		// 检查是否已存在
		s.mutex.RLock()
		_, exists := s.sessions[code]
		s.mutex.RUnlock()

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after 10 attempts")
}

// cleanupExpiredSessions 清理过期会话
func (s *VerifyCodeService) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		s.mutex.Lock()
		now := time.Now()
		for code, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, code)
				s.logger.Debug("Cleaned up expired session",
					zap.String("code", code),
					zap.Uint64("account_id", session.AccountID))
			}
		}
		s.mutex.Unlock()
	}
}

// GetSessionInfo 获取会话信息 (用于调试)
func (s *VerifyCodeService) GetSessionInfo(code string) *models.VerifyCodeSession {
	s.mutex.RLock()
	session, exists := s.sessions[code]
	s.mutex.RUnlock()

	if !exists {
		return nil
	}

	// 返回副本
	sessionCopy := *session
	return &sessionCopy
}

// verifyCodeTaskResult 验证码任务结果
type verifyCodeTaskResult struct {
	success    bool
	code       string
	sender     string
	receivedAt time.Time
}

// verifyCodeTask 验证码获取任务
type verifyCodeTask struct {
	timeoutSeconds int
	logger         *zap.Logger
	result         verifyCodeTaskResult
}

// Execute 实现 TaskInterface.Execute
func (t *verifyCodeTask) Execute(ctx context.Context, api *tg.Client) error {
	// 验证码发送者白名单
	senders := []string{"777000", "Telegram"}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.timeoutSeconds)*time.Second)
	defer cancel()

	startTime := time.Now()

	// 轮询检查新消息
	for {
		select {
		case <-timeoutCtx.Done():
			t.result = verifyCodeTaskResult{success: false}
			return nil
		default:
			// 获取最新对话
			dialogs, err := api.MessagesGetDialogs(timeoutCtx, &tg.MessagesGetDialogsRequest{
				Limit: 20,
			})
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			// 检查每个对话的最新消息
			if code, sender, receivedTime, found := t.searchVerifyCode(dialogs, senders, startTime); found {
				t.result = verifyCodeTaskResult{
					success:    true,
					code:       code,
					sender:     sender,
					receivedAt: receivedTime,
				}
				return nil
			}

			// 等待2秒后再次检查
			time.Sleep(2 * time.Second)
		}
	}
}

// GetType 实现 TaskInterface.GetType
func (t *verifyCodeTask) GetType() string {
	return "verify_code_retrieval"
}

// searchVerifyCode 在对话中搜索验证码
func (t *verifyCodeTask) searchVerifyCode(dialogs tg.MessagesDialogsClass, senders []string, startTime time.Time) (code, sender string, receivedTime time.Time, found bool) {
	if messagesDialogs, ok := dialogs.(*tg.MessagesDialogs); ok {
		for _, message := range messagesDialogs.Messages {
			if msg, ok := message.(*tg.Message); ok {
				// 检查消息时间是否在开始时间后
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
func (t *verifyCodeTask) extractVerificationCode(message string) string {
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
func (t *verifyCodeTask) containsIgnoreCase(text, pattern string) bool {
	textLower := t.toLowerCase(text)
	patternLower := t.toLowerCase(pattern)
	return t.contains(textLower, patternLower)
}

// toLowerCase 转换为小写
func (t *verifyCodeTask) toLowerCase(str string) string {
	result := make([]rune, len(str))
	for i, r := range str {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// contains 检查字符串是否包含子字符串
func (t *verifyCodeTask) contains(str, substr string) bool {
	if len(substr) > len(str) {
		return false
	}

	for i := 0; i <= len(str)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if str[i+j] != substr[j] {
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
