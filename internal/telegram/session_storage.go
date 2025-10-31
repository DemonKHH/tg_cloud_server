package telegram

import (
	"context"

	"github.com/gotd/td/session"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/repository"
)

// DatabaseSessionStorage 基于数据库的Session存储
type DatabaseSessionStorage struct {
	accountID  uint64
	accountRepo repository.AccountRepository
	data      []byte
	logger    *zap.Logger
}

// NewDatabaseSessionStorage 创建数据库Session存储
func NewDatabaseSessionStorage(accountID uint64, accountRepo repository.AccountRepository, sessionData []byte) *DatabaseSessionStorage {
	return &DatabaseSessionStorage{
		accountID:  accountID,
		accountRepo: accountRepo,
		data:       sessionData,
		logger:     logger.Get().Named("session_storage"),
	}
}

// LoadSession 加载Session数据
func (s *DatabaseSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	// 如果内存中有数据，直接返回（优先使用）
	if s.data != nil && len(s.data) > 0 {
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

	// 如果数据库中有session数据，加载并缓存
	if account.SessionData != "" {
		sessionData := []byte(account.SessionData)
		s.data = sessionData // 缓存到内存
		s.logger.Debug("Loaded session from database",
			zap.Uint64("account_id", s.accountID),
			zap.Int("data_len", len(sessionData)))
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

	// 保存到数据库
	err := s.accountRepo.UpdateSessionData(s.accountID, data)
	if err != nil {
		s.logger.Error("Failed to save session to database",
			zap.Uint64("account_id", s.accountID),
			zap.Error(err))
		return err
	}

	s.logger.Debug("Session saved to database",
		zap.Uint64("account_id", s.accountID),
		zap.Int("data_len", len(data)))
	return nil
}
