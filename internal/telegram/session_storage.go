package telegram

import (
	"context"
	"encoding/base64"

	"github.com/gotd/td/session"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/repository"
)

// DatabaseSessionStorage 基于数据库的Session存储
type DatabaseSessionStorage struct {
	accountID   uint64
	accountRepo repository.AccountRepository
	data        []byte
	logger      *zap.Logger
}

// NewDatabaseSessionStorage 创建数据库Session存储
func NewDatabaseSessionStorage(accountID uint64, accountRepo repository.AccountRepository, sessionData []byte) *DatabaseSessionStorage {
	return &DatabaseSessionStorage{
		accountID:   accountID,
		accountRepo: accountRepo,
		data:        sessionData,
		logger:      logger.Get().Named("session_storage"),
	}
}

// LoadSession 加载Session数据
func (s *DatabaseSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	// 如果内存中有数据，直接返回（优先使用）
	if s.data != nil {
		s.logger.Debug("Loading session from memory",
			zap.Uint64("account_id", s.accountID),
			zap.Int("data_len", len(s.data)))
		return s.data, nil
	}

	// 从数据库加载session数据
	account, err := s.accountRepo.GetByID(s.accountID)
	if err != nil {
		s.logger.Warn("Failed to load account from database",
			zap.Uint64("account_id", s.accountID),
			zap.Error(err))
		return nil, session.ErrNotFound
	}

	// 如果数据库中有session数据，解码并缓存
	if account.SessionData != "" {
		// 数据库存储的是base64编码的gotd JSON格式session
		// 解码后得到JSON字符串，gotd期望收到JSON字符串的[]byte
		sessionData, err := base64.StdEncoding.DecodeString(account.SessionData)
		if err != nil {
			s.logger.Error("Failed to decode session data from database",
				zap.Uint64("account_id", s.accountID),
				zap.String("session_data_preview", account.SessionData[:min(50, len(account.SessionData))]),
				zap.Error(err))
			return nil, session.ErrNotFound
		}

		s.data = sessionData // 缓存到内存
		s.logger.Debug("Loaded gotd session from database",
			zap.Uint64("account_id", s.accountID),
			zap.Int("json_data_len", len(sessionData)))
		return sessionData, nil
	}

	// 没有session数据
	s.logger.Debug("No session data found",
		zap.Uint64("account_id", s.accountID))
	return nil, session.ErrNotFound
}

// StoreSession 存储Session数据
func (s *DatabaseSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	// 更新内存缓存
	s.data = data

	// gotd传入的data是JSON格式的session数据，将其编码为base64字符串存储
	encodedData := base64.StdEncoding.EncodeToString(data)
	err := s.accountRepo.UpdateSessionData(s.accountID, []byte(encodedData))
	if err != nil {
		s.logger.Error("Failed to save gotd session to database",
			zap.Uint64("account_id", s.accountID),
			zap.Error(err))
		return err
	}

	s.logger.Debug("Gotd session encoded and saved to database",
		zap.Uint64("account_id", s.accountID),
		zap.Int("json_data_len", len(data)),
		zap.Int("encoded_data_len", len(encodedData)))
	return nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
