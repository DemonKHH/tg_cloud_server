package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/config"
	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
)

// AuthService 认证服务
type AuthService struct {
	userRepo repository.UserRepository
	config   *config.Config
	logger   *zap.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repository.UserRepository, config *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		config:   config,
		logger:   logger.Get().Named("auth_service"),
	}
}

// Register 用户注册
func (s *AuthService) Register(req *models.RegisterRequest) (*models.UserProfile, error) {
	// 检查用户名是否已存在
	existingUser, _ := s.userRepo.GetByUsername(req.Username)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 检查邮箱是否已存在
	existingUser, _ = s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// 创建新用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     models.RoleStandard,
		IsActive: true,
	}

	// 设置密码哈希
	if err := user.SetPassword(req.Password); err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("password processing failed: %w", err)
	}

	// 保存用户到数据库
	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("user creation failed: %w", err)
	}

	// 生成用户统计信息
	stats, err := s.generateUserStats(user.ID)
	if err != nil {
		s.logger.Warn("Failed to generate user stats",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
		stats = &models.UserStats{}
	}

	// 构建用户资料
	profile := &models.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		IsActive:    user.IsActive,
		IsExpired:   user.IsExpired(),
		ExpiresAt:   user.ExpiresAt,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Stats:       *stats,
	}

	s.logger.Info("User registered successfully",
		zap.Uint64("user_id", user.ID),
		zap.String("username", user.Username))

	return profile, nil
}

// Login 用户登录
func (s *AuthService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// 获取用户
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// 检查用户是否过期
	if user.IsExpired() {
		return nil, models.NewUserExpiredError(user)
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		s.logger.Warn("Failed to update last login time",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
	}

	// 生成访问令牌
	accessToken, expiresIn, err := s.generateAccessToken(user)
	if err != nil {
		s.logger.Error("Failed to generate access token",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	// 生成用户统计信息
	stats, err := s.generateUserStats(user.ID)
	if err != nil {
		s.logger.Warn("Failed to generate user stats",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
		stats = &models.UserStats{}
	}

	// 构建用户资料
	userProfile := models.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		IsActive:    user.IsActive,
		IsExpired:   user.IsExpired(),
		ExpiresAt:   user.ExpiresAt,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Stats:       *stats,
	}

	response := &models.LoginResponse{
		User:        userProfile,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	s.logger.Info("User logged in successfully",
		zap.Uint64("user_id", user.ID),
		zap.String("username", user.Username))

	return response, nil
}

// GetUserProfile 获取用户资料
func (s *AuthService) GetUserProfile(userID uint64) (*models.UserProfile, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 生成用户统计信息
	stats, err := s.generateUserStats(userID)
	if err != nil {
		s.logger.Warn("Failed to generate user stats",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		stats = &models.UserStats{}
	}

	profile := &models.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		IsActive:    user.IsActive,
		IsExpired:   user.IsExpired(),
		ExpiresAt:   user.ExpiresAt,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Stats:       *stats,
	}

	return profile, nil
}

// UpdateUserProfile 更新用户资料
func (s *AuthService) UpdateUserProfile(userID uint64, req *models.UpdateProfileRequest) (*models.UserProfile, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 更新邮箱
	if req.Email != "" {
		// 检查邮箱是否被其他用户使用
		existingUser, _ := s.userRepo.GetByEmail(req.Email)
		if existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("email already in use")
		}
		user.Email = req.Email
	}

	// 更新密码
	if req.Password != "" {
		if err := user.SetPassword(req.Password); err != nil {
			s.logger.Error("Failed to hash new password", zap.Error(err))
			return nil, fmt.Errorf("password update failed: %w", err)
		}
	}

	// 保存更改
	if err := s.userRepo.Update(user); err != nil {
		s.logger.Error("Failed to update user",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		return nil, fmt.Errorf("user update failed: %w", err)
	}

	s.logger.Info("User profile updated successfully",
		zap.Uint64("user_id", userID))

	return s.GetUserProfile(userID)
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(refreshToken string) (*models.LoginResponse, error) {
	// 解析刷新令牌
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 获取用户
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(uint64(userID))
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// 检查用户是否过期
	if user.IsExpired() {
		return nil, models.NewUserExpiredError(user)
	}

	// 生成新的访问令牌
	accessToken, expiresIn, err := s.generateAccessToken(user)
	if err != nil {
		s.logger.Error("Failed to generate new access token",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
		return nil, fmt.Errorf("token generation failed: %w", err)
	}

	// 生成用户统计信息
	stats, err := s.generateUserStats(user.ID)
	if err != nil {
		s.logger.Warn("Failed to generate user stats",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
		stats = &models.UserStats{}
	}

	userProfile := models.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		IsActive:    user.IsActive,
		IsExpired:   user.IsExpired(),
		ExpiresAt:   user.ExpiresAt,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		Stats:       *stats,
	}

	response := &models.LoginResponse{
		User:        userProfile,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	return response, nil
}

// Logout 用户登出
func (s *AuthService) Logout(userID uint64, token string) error {
	// 这里可以将token加入黑名单
	// 实际实现中应该使用Redis存储黑名单令牌

	s.logger.Info("User logged out", zap.Uint64("user_id", userID))
	return nil
}

// VerifyToken 验证访问令牌
func (s *AuthService) VerifyToken(tokenString string) (uint64, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return 0, ErrInvalidToken
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, ErrInvalidToken
	}

	return uint64(userID), nil
}

// generateAccessToken 生成访问令牌
func (s *AuthService) generateAccessToken(user *models.User) (string, int64, error) {
	// 设置过期时间
	expirationTime := time.Now().Add(s.config.JWT.ExpirationTime)

	// 创建claims
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expirationTime.Unix(),
		"iat":      time.Now().Unix(),
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名token
	tokenString, err := token.SignedString([]byte(s.config.JWT.SecretKey))
	if err != nil {
		return "", 0, err
	}

	expiresIn := int64(s.config.JWT.ExpirationTime.Seconds())
	return tokenString, expiresIn, nil
}

// parseToken 解析令牌
func (s *AuthService) parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// generateUserStats 生成用户统计信息
func (s *AuthService) generateUserStats(userID uint64) (*models.UserStats, error) {
	// 这里应该调用相应的repository方法获取统计数据
	// 为了简化示例，返回默认值
	stats := &models.UserStats{
		AccountCount:       0,
		ActiveAccountCount: 0,
		TaskCount:          0,
		TasksToday:         0,
		TasksThisWeek:      0,
		ProxyCount:         0,
	}

	// TODO: 实现实际的统计查询
	// stats.AccountCount = s.accountRepo.CountByUserID(userID)
	// stats.TaskCount = s.taskRepo.CountByUserID(userID)
	// etc.

	return stats, nil
}
