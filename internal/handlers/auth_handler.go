package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
	logger      *zap.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger.Get().Named("auth_handler"),
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "注册信息"
// @Success 201 {object} models.UserProfile "用户信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 409 {object} map[string]string "用户已存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid register request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "请求参数无效",
			"details": err.Error(),
		})
		return
	}

	// 调用服务层注册用户
	user, err := h.authService.Register(&req)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "user_exists",
				"message": "用户名或邮箱已存在",
			})
			return
		}

		h.logger.Error("Register failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "register_failed",
			"message": "注册失败，请稍后重试",
		})
		return
	}

	h.logger.Info("User registered successfully", 
		zap.String("username", user.Username),
		zap.Uint64("user_id", user.ID))

	c.JSON(http.StatusCreated, user)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "登录信息"
// @Success 200 {object} models.LoginResponse "登录成功"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "认证失败"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "请求参数无效",
			"details": err.Error(),
		})
		return
	}

	// 调用服务层登录
	response, err := h.authService.Login(&req)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_credentials",
				"message": "用户名或密码错误",
			})
			return
		}

		h.logger.Error("Login failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "login_failed",
			"message": "登录失败，请稍后重试",
		})
		return
	}

	h.logger.Info("User logged in successfully", 
		zap.String("username", req.Username))

	c.JSON(http.StatusOK, response)
}

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.UserProfile "用户资料"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从JWT中间件获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "未找到用户信息",
		})
		return
	}

	uid, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "用户ID格式错误",
		})
		return
	}

	// 获取用户资料
	profile, err := h.authService.GetUserProfile(uid)
	if err != nil {
		h.logger.Error("Failed to get user profile", 
			zap.Uint64("user_id", uid),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "profile_failed",
			"message": "获取用户资料失败",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前登录用户的资料信息
// @Tags 认证
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.UpdateProfileRequest true "更新信息"
// @Success 200 {object} models.UserProfile "更新后的用户资料"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "未找到用户信息",
		})
		return
	}

	uid, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "用户ID格式错误",
		})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update profile request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "请求参数无效",
			"details": err.Error(),
		})
		return
	}

	// 更新用户资料
	profile, err := h.authService.UpdateUserProfile(uid, &req)
	if err != nil {
		h.logger.Error("Failed to update user profile", 
			zap.Uint64("user_id", uid),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "update_failed",
			"message": "更新用户资料失败",
		})
		return
	}

	h.logger.Info("User profile updated successfully", 
		zap.Uint64("user_id", uid))

	c.JSON(http.StatusOK, profile)
}

// RefreshToken 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param refresh_token header string true "刷新令牌"
// @Success 200 {object} models.LoginResponse "新的访问令牌"
// @Failure 401 {object} map[string]string "令牌无效"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("Refresh-Token")
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_token",
			"message": "缺少刷新令牌",
		})
		return
	}

	// 刷新令牌
	response, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		if err == services.ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": "无效的刷新令牌",
			})
			return
		}

		h.logger.Error("Token refresh failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "refresh_failed",
			"message": "令牌刷新失败",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出，使令牌失效
// @Tags 认证
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string "登出成功"
// @Failure 401 {object} map[string]string "未授权"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "未找到用户信息",
		})
		return
	}

	uid, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "用户ID格式错误",
		})
		return
	}

	// 获取令牌
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_token",
			"message": "缺少访问令牌",
		})
		return
	}

	// 执行登出
	if err := h.authService.Logout(uid, token); err != nil {
		h.logger.Error("Logout failed", 
			zap.Uint64("user_id", uid),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "logout_failed",
			"message": "登出失败",
		})
		return
	}

	h.logger.Info("User logged out successfully", 
		zap.Uint64("user_id", uid))

	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}
