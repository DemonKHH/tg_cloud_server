package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	accountParser  *services.AccountParser
	logger         *zap.Logger
}

// NewAccountHandler 创建账号管理处理器
func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		accountParser:  services.NewAccountParser(),
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
// @Param search query string false "搜索关键词（手机号或备注）"
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
	search := c.Query("search")

	// 构建过滤器
	filter := &services.AccountFilter{
		UserID: userID,
		Status: status,
		Search: search,
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

// UploadAccountFiles 批量上传账号信息
// @Summary 批量上传账号信息
// @Description 批量上传Telegram账号信息，支持文件上传（zip、.session、tdata）或直接上传JSON数据
// @Tags 账号管理
// @Accept multipart/form-data,application/json
// @Produce json
// @Security ApiKeyAuth
// @Param file formData file false "账号文件（zip、.session或tdata文件夹）"
// @Param request body models.BatchUploadAccountRequest false "批量账号信息（JSON格式，与file二选一）"
// @Param proxy_id formData string false "代理ID"
// @Success 200 {object} map[string]interface{} "上传结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/upload [post]
func (h *AccountHandler) UploadAccountFiles(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	// 获取代理ID（可选）
	var proxyID *uint64
	if proxyIDStr := c.PostForm("proxy_id"); proxyIDStr != "" {
		if id, err := strconv.ParseUint(proxyIDStr, 10, 64); err == nil {
			proxyID = &id
		}
	}

	// 检查是否是文件上传
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		// 文件上传模式
		defer file.Close()
		h.handleFileUpload(c, userID, file, header, proxyID)
		return
	}

	// JSON 上传模式（向后兼容）
	var req models.BatchUploadAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("解析请求失败", zap.Error(err))
		response.InvalidParam(c, "请求参数错误: "+err.Error())
		return
	}

	if len(req.Accounts) == 0 {
		response.InvalidParam(c, "账号列表不能为空")
		return
	}

	// 使用请求中的proxy_id，如果没有则使用form中的
	if req.ProxyID == nil {
		req.ProxyID = proxyID
	}

	// 批量创建账号
	createdAccounts, errors, err := h.accountService.CreateAccountsFromUploadData(userID, req.Accounts, req.ProxyID)
	if err != nil {
		h.logger.Error("批量创建账号失败", zap.Error(err))
		response.InternalError(c, "创建账号失败: "+err.Error())
		return
	}

	result := gin.H{
		"total":    len(req.Accounts),
		"created":  len(createdAccounts),
		"failed":   len(errors),
		"accounts": createdAccounts,
		"errors":   errors,
	}

	if len(errors) > 0 {
		h.logger.Warn("部分账号创建失败",
			zap.Int("total", len(req.Accounts)),
			zap.Int("created", len(createdAccounts)),
			zap.Int("failed", len(errors)))
	}

	h.logger.Info("账号批量上传完成",
		zap.Uint64("user_id", userID),
		zap.Int("total", len(req.Accounts)),
		zap.Int("created", len(createdAccounts)),
		zap.Int("failed", len(errors)))

	response.SuccessWithMessage(c, fmt.Sprintf("成功创建 %d 个账号，失败 %d 个", len(createdAccounts), len(errors)), result)
}

// BatchSet2FA 批量设置2FA密码
// @Summary 批量设置2FA密码
// @Description 批量设置账号的2FA密码（仅更新本地记录）
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.BatchSet2FARequest true "设置信息"
// @Success 200 {object} map[string]string "操作成功"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/batch/set-2fa [post]
func (h *AccountHandler) BatchSet2FA(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.BatchSet2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch set 2fa request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	h.logger.Info("Batch setting 2FA passwords",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(req.AccountIDs)))

	if err := h.accountService.BatchSet2FA(userID, &req); err != nil {
		h.logger.Error("Failed to batch set 2fa",
			zap.Uint64("user_id", userID),
			zap.Int("account_count", len(req.AccountIDs)),
			zap.Error(err))
		response.InternalError(c, "批量设置2FA失败")
		return
	}

	h.logger.Info("Batch 2FA passwords set successfully",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(req.AccountIDs)))

	response.SuccessWithMessage(c, "批量设置2FA密码成功", nil)
}

// BatchUpdate2FA 批量修改2FA密码
// @Summary 批量修改2FA密码
// @Description 批量修改账号的2FA密码（尝试修改Telegram密码）
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.BatchUpdate2FARequest true "修改信息"
// @Success 200 {object} map[string]interface{} "操作结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/batch/update-2fa [post]
func (h *AccountHandler) BatchUpdate2FA(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.BatchUpdate2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch update 2fa request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	results, err := h.accountService.BatchUpdate2FA(userID, &req)
	if err != nil {
		h.logger.Error("Failed to batch update 2fa", zap.Error(err))
		response.InternalError(c, "批量修改2FA失败")
		return
	}

	response.SuccessWithMessage(c, "批量修改2FA操作完成", results)
}

// BatchDeleteAccounts 批量删除账号
// @Summary 批量删除账号
// @Description 批量删除指定的TG账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.BatchDeleteAccountsRequest true "删除信息"
// @Success 200 {object} map[string]interface{} "操作结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/batch/delete [post]
func (h *AccountHandler) BatchDeleteAccounts(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.BatchDeleteAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch delete accounts request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	if len(req.AccountIDs) == 0 {
		response.InvalidParam(c, "账号ID列表不能为空")
		return
	}

	h.logger.Info("Batch deleting accounts",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(req.AccountIDs)))

	successCount, failedCount, err := h.accountService.BatchDeleteAccounts(userID, req.AccountIDs)
	if err != nil {
		h.logger.Error("Failed to batch delete accounts",
			zap.Uint64("user_id", userID),
			zap.Int("account_count", len(req.AccountIDs)),
			zap.Error(err))
		response.InternalError(c, "批量删除账号失败")
		return
	}

	h.logger.Info("Batch delete accounts completed",
		zap.Uint64("user_id", userID),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount))

	response.SuccessWithMessage(c, fmt.Sprintf("成功删除 %d 个账号，失败 %d 个", successCount, failedCount), gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
	})
}

// BatchBindProxy 批量绑定/解绑代理
// @Summary 批量绑定/解绑代理
// @Description 批量为账号绑定或解绑代理，proxy_id为null时表示解绑
// @Tags 账号管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.BatchBindProxyRequest true "绑定信息"
// @Success 200 {object} map[string]interface{} "操作结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/batch/bind-proxy [post]
func (h *AccountHandler) BatchBindProxy(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.BatchBindProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch bind proxy request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	if len(req.AccountIDs) == 0 {
		response.InvalidParam(c, "账号ID列表不能为空")
		return
	}

	action := "绑定"
	if req.ProxyID == nil {
		action = "解绑"
	}

	h.logger.Info("Batch binding proxy",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(req.AccountIDs)),
		zap.Any("proxy_id", req.ProxyID),
		zap.String("action", action))

	successCount, failedCount, err := h.accountService.BatchBindProxy(userID, req.AccountIDs, req.ProxyID)
	if err != nil {
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}
		h.logger.Error("Failed to batch bind proxy",
			zap.Uint64("user_id", userID),
			zap.Int("account_count", len(req.AccountIDs)),
			zap.Error(err))
		response.InternalError(c, "批量"+action+"代理失败: "+err.Error())
		return
	}

	h.logger.Info("Batch bind proxy completed",
		zap.Uint64("user_id", userID),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
		zap.String("action", action))

	response.SuccessWithMessage(c, fmt.Sprintf("成功%s %d 个账号的代理，失败 %d 个", action, successCount, failedCount), gin.H{
		"success_count": successCount,
		"failed_count":  failedCount,
	})
}

// handleFileUpload 处理文件上传
func (h *AccountHandler) handleFileUpload(c *gin.Context, userID uint64, file multipart.File, header *multipart.FileHeader, proxyID *uint64) {
	h.logger.Info("Processing file upload",
		zap.Uint64("user_id", userID),
		zap.String("filename", header.Filename),
		zap.Int64("file_size", header.Size),
		zap.Any("proxy_id", proxyID))

	// 验证文件大小（100MB限制）
	if header.Size > 100*1024*1024 {
		h.logger.Warn("File size exceeds limit",
			zap.Uint64("user_id", userID),
			zap.String("filename", header.Filename),
			zap.Int64("file_size", header.Size))
		response.InvalidParam(c, "文件大小超过100MB限制")
		return
	}

	// 创建临时文件
	tempDir, err := os.MkdirTemp("", "account_upload_*")
	if err != nil {
		h.logger.Error("创建临时目录失败",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "创建临时目录失败")
		return
	}
	defer os.RemoveAll(tempDir)

	// 保存上传的文件
	fileName := header.Filename
	tempFilePath := filepath.Join(tempDir, fileName)

	dst, err := os.Create(tempFilePath)
	if err != nil {
		h.logger.Error("创建临时文件失败", zap.Error(err))
		response.InternalError(c, "创建临时文件失败")
		return
	}

	_, err = io.Copy(dst, file)
	dst.Close()
	if err != nil {
		h.logger.Error("保存文件失败", zap.Error(err))
		response.InternalError(c, "保存文件失败")
		return
	}

	// 解析账号文件
	parsedAccounts, err := h.accountParser.ParseAccountFiles(tempFilePath)
	if err != nil {
		h.logger.Error("解析账号文件失败", zap.Error(err))
		response.InvalidParam(c, "解析账号文件失败: "+err.Error())
		return
	}

	if len(parsedAccounts) == 0 {
		response.InvalidParam(c, "未能从文件中解析出账号信息")
		return
	}

	// 转换为上传数据格式
	var uploadItems []models.AccountUploadItem
	var parseErrors []string

	for _, account := range parsedAccounts {
		if account.Error != "" {
			parseErrors = append(parseErrors, fmt.Sprintf("账号 %s: %s", account.Phone, account.Error))
			continue
		}

		if account.Phone == "" || account.SessionData == "" {
			parseErrors = append(parseErrors, fmt.Sprintf("账号数据不完整: Phone=%s", account.Phone))
			continue
		}

		uploadItems = append(uploadItems, models.AccountUploadItem{
			Phone:       account.Phone,
			SessionData: account.SessionData,
		})
	}

	if len(uploadItems) == 0 {
		response.InvalidParam(c, "未能解析出有效的账号信息")
		return
	}

	// 批量创建账号
	createdAccounts, createErrors, err := h.accountService.CreateAccountsFromUploadData(userID, uploadItems, proxyID)
	if err != nil {
		h.logger.Error("批量创建账号失败", zap.Error(err))
		response.InternalError(c, "创建账号失败: "+err.Error())
		return
	}

	// 合并解析错误和创建错误
	allErrors := append(parseErrors, createErrors...)

	result := gin.H{
		"total":    len(parsedAccounts),
		"created":  len(createdAccounts),
		"failed":   len(allErrors),
		"accounts": createdAccounts,
	}

	if len(allErrors) > 0 {
		result["errors"] = allErrors
		h.logger.Warn("部分账号创建失败",
			zap.Int("total", len(parsedAccounts)),
			zap.Int("created", len(createdAccounts)),
			zap.Int("failed", len(allErrors)))
	}

	h.logger.Info("账号文件上传完成",
		zap.Uint64("user_id", userID),
		zap.String("file", fileName),
		zap.Int("total", len(parsedAccounts)),
		zap.Int("created", len(createdAccounts)),
		zap.Int("failed", len(allErrors)))

	response.SuccessWithMessage(c, fmt.Sprintf("成功创建 %d 个账号，失败 %d 个", len(createdAccounts), len(allErrors)), result)
}

// ExportAccounts 导出账号
// @Summary 导出账号
// @Description 导出选中的账号为zip文件，每个账号一个文件夹，包含session文件
// @Tags 账号管理
// @Accept json
// @Produce application/zip
// @Security ApiKeyAuth
// @Param request body models.ExportAccountsRequest true "导出请求"
// @Success 200 {file} file "zip文件"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/accounts/export [post]
func (h *AccountHandler) ExportAccounts(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == 0 {
		return
	}

	var req models.ExportAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid export accounts request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	if len(req.AccountIDs) == 0 {
		response.InvalidParam(c, "账号ID列表不能为空")
		return
	}

	h.logger.Info("Exporting accounts",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(req.AccountIDs)))

	// 获取账号数据
	accounts, err := h.accountService.GetAccountsForExport(userID, req.AccountIDs)
	if err != nil {
		h.logger.Error("Failed to get accounts for export",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取账号数据失败")
		return
	}

	if len(accounts) == 0 {
		response.InvalidParam(c, "没有找到可导出的账号")
		return
	}

	// 创建zip文件
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	exportedCount := 0
	for _, account := range accounts {
		if account.SessionData == "" {
			h.logger.Warn("Account has no session data, skipping",
				zap.Uint64("account_id", account.ID),
				zap.String("phone", account.Phone))
			continue
		}

		// 解码base64的session数据
		// 数据库中存储的是base64编码的gotd JSON格式session
		sessionBytes, err := base64.StdEncoding.DecodeString(account.SessionData)
		if err != nil {
			h.logger.Error("Failed to decode session data",
				zap.String("phone", account.Phone),
				zap.Error(err))
			continue
		}

		// 创建文件夹路径: 手机号/手机号.session
		folderPath := account.Phone + "/"
		sessionFileName := folderPath + account.Phone + ".session"

		// 创建session文件
		fileWriter, err := zipWriter.Create(sessionFileName)
		if err != nil {
			h.logger.Error("Failed to create file in zip",
				zap.String("phone", account.Phone),
				zap.Error(err))
			continue
		}

		// 写入解码后的session数据（二进制格式）
		_, err = fileWriter.Write(sessionBytes)
		if err != nil {
			h.logger.Error("Failed to write session data",
				zap.String("phone", account.Phone),
				zap.Error(err))
			continue
		}

		exportedCount++
	}

	// 关闭zip writer
	if err := zipWriter.Close(); err != nil {
		h.logger.Error("Failed to close zip writer", zap.Error(err))
		response.InternalError(c, "创建zip文件失败")
		return
	}

	if exportedCount == 0 {
		response.InvalidParam(c, "没有可导出的账号数据")
		return
	}

	h.logger.Info("Accounts exported successfully",
		zap.Uint64("user_id", userID),
		zap.Int("exported_count", exportedCount))

	// 设置响应头
	fileName := fmt.Sprintf("accounts_export_%s.zip", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	// 发送文件
	c.Data(200, "application/zip", buf.Bytes())
}
