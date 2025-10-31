package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// AccountHandler 账号管理处理器
type AccountHandler struct {
	accountService *services.AccountService
	logger         *zap.Logger
}

// NewAccountHandler 创建账号管理处理器
func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		logger:         logger.Get().Named("account_handler"),
	}
}

// CreateAccount 添加TG账号
// @Summary 添加TG账号
// @Description 添加新的Telegram账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.CreateAccountRequest true "账号信息"
// @Success 201 {object} models.TGAccount "创建成功的账号"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 409 {object} map[string]string "账号已存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create account request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 创建账号
	account, err := h.accountService.CreateAccount(userID, &req)
	if err != nil {
		if err == services.ErrAccountExists {
			response.Conflict(c, "该手机号已存在")
			return
		}

		h.logger.Error("Failed to create account",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "创建账号失败")
		return
	}

	h.logger.Info("Account created successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", account.ID),
		zap.String("phone", account.Phone))

	response.SuccessWithMessage(c, "账号创建成功", account)
}

// GetAccounts 获取账号列表
// @Summary 获取账号列表
// @Description 获取当前用户的所有TG账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param status query string false "账号状态过滤"
// @Success 200 {object} models.PaginationResponse "账号列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts [get]
func (h *AccountHandler) GetAccounts(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	// 解析查询参数
	page := h.getIntParam(c, "page", 1)
	limit := h.getIntParam(c, "limit", 20)
	status := c.Query("status")

	// 构建过滤器
	filter := &services.AccountFilter{
		UserID: userID,
		Status: status,
		Page:   page,
		Limit:  limit,
	}

	// 获取账号列表
	accounts, total, err := h.accountService.GetAccounts(filter)
	if err != nil {
		h.logger.Error("Failed to get accounts",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取账号列表失败")
		return
	}

	response.Paginated(c, accounts, page, limit, total)
}

// GetAccount 获取账号详情
// @Summary 获取账号详情
// @Description 获取指定TG账号的详细信息
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Success 200 {object} models.TGAccount "账号详情"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	// 获取账号详情
	account, err := h.accountService.GetAccount(userID, accountID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}

		h.logger.Error("Failed to get account",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, "获取账号详情失败")
		return
	}

	response.Success(c, account)
}

// UpdateAccount 更新账号信息
// @Summary 更新账号信息
// @Description 更新指定TG账号的信息
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Param request body models.UpdateAccountRequest true "更新信息"
// @Success 200 {object} models.TGAccount "更新后的账号"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	var req models.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update account request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 更新账号
	account, err := h.accountService.UpdateAccount(userID, accountID, &req)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}

		h.logger.Error("Failed to update account",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, "更新账号失败")
		return
	}

	h.logger.Info("Account updated successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID))

	response.SuccessWithMessage(c, "账号更新成功", account)
}

// DeleteAccount 删除账号
// @Summary 删除账号
// @Description 删除指定的TG账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	// 删除账号
	err := h.accountService.DeleteAccount(userID, accountID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}

		h.logger.Error("Failed to delete account",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, "删除账号失败")
		return
	}

	h.logger.Info("Account deleted successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID))

	response.SuccessWithMessage(c, "账号删除成功", nil)
}

// CheckAccountHealth 检查账号健康度
// @Summary 检查账号健康度
// @Description 检查指定TG账号的健康状态
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Success 200 {object} models.AccountHealthReport "健康度报告"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id}/health [get]
func (h *AccountHandler) CheckAccountHealth(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	// 检查账号健康度
	report, err := h.accountService.CheckAccountHealth(userID, accountID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}

		h.logger.Error("Failed to check account health",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, "健康度检查失败")
		return
	}

	response.Success(c, report)
}

// GetAccountAvailability 获取账号可用性
// @Summary 获取账号可用性
// @Description 获取指定账号的可用性信息（用于任务分配）
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Success 200 {object} models.AccountAvailability "可用性信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id}/availability [get]
func (h *AccountHandler) GetAccountAvailability(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	// 获取账号可用性
	availability, err := h.accountService.GetAccountAvailability(userID, accountID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}

		h.logger.Error("Failed to get account availability",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, "获取可用性失败")
		return
	}

	response.Success(c, availability)
}

// BindProxy 绑定代理到账号
// @Summary 绑定代理到账号
// @Description 为指定账号绑定代理IP
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "账号ID"
// @Param request body models.BindProxyRequest true "代理绑定信息"
// @Success 200 {object} models.TGAccount "绑定成功的账号"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号或代理不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/{id}/bind-proxy [post]
func (h *AccountHandler) BindProxy(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	accountID := h.getIDParam(c, "id")
	if accountID == 0 {
		return
	}

	var req models.BindProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid bind proxy request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 绑定代理
	account, err := h.accountService.BindProxy(userID, accountID, req.ProxyID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			response.AccountNotFound(c)
			return
		}
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}

		h.logger.Error("Failed to bind proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Any("proxy_id", req.ProxyID),
			zap.Error(err))
		response.InternalError(c, "代理绑定失败")
		return
	}

	h.logger.Info("Proxy bound successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID),
		zap.Any("proxy_id", req.ProxyID))

	response.SuccessWithMessage(c, "代理绑定成功", account)
}

// 辅助方法

// getUserID 从上下文获取用户ID
func (h *AccountHandler) getUserID(c *gin.Context) uint64 {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未找到用户信息")
		return 0
	}

	uid, ok := userID.(uint64)
	if !ok {
		response.Unauthorized(c, "用户ID格式错误")
		return 0
	}

	return uid
}

// getIDParam 获取路径参数中的ID
func (h *AccountHandler) getIDParam(c *gin.Context, param string) uint64 {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的ID参数")
		return 0
	}
	return id
}

// getIntParam 获取查询参数中的整数值
func (h *AccountHandler) getIntParam(c *gin.Context, param string, defaultValue int) int {
	valueStr := c.Query(param)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
