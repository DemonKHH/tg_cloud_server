package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// ProxyHandler 代理处理器
type ProxyHandler struct {
	proxyService services.ProxyService
	logger       *zap.Logger
}

// NewProxyHandler 创建代理处理器
func NewProxyHandler(proxyService services.ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		logger:       logger.Get().Named("proxy_handler"),
	}
}

// CreateProxy 创建代理
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	proxy, err := h.proxyService.CreateProxy(userID, &req)
	if err != nil {
		h.logger.Error("Failed to create proxy",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "代理创建成功", proxy)
}

// GetProxies 获取代理列表
func (h *ProxyHandler) GetProxies(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	status := c.Query("status")
	var proxies []*models.ProxyIP
	var total int64

	if status != "" {
		proxies, total, err = h.proxyService.GetProxiesByStatus(userID, status, page, limit)
	} else {
		proxies, total, err = h.proxyService.GetProxies(userID, page, limit)
	}

	if err != nil {
		h.logger.Error("Failed to get proxies",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取代理列表失败")
		return
	}

	response.Paginated(c, proxies, page, limit, total)
}

// GetProxy 获取代理详情
func (h *ProxyHandler) GetProxy(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的代理ID")
		return
	}

	proxy, err := h.proxyService.GetProxy(userID, proxyID)
	if err != nil {
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}
		h.logger.Error("Failed to get proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		response.InternalError(c, "获取代理失败")
		return
	}

	response.Success(c, proxy)
}

// UpdateProxy 更新代理
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的代理ID")
		return
	}

	var req models.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	proxy, err := h.proxyService.UpdateProxy(userID, proxyID, &req)
	if err != nil {
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}
		h.logger.Error("Failed to update proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "代理更新成功", proxy)
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的代理ID")
		return
	}

	if err := h.proxyService.DeleteProxy(userID, proxyID); err != nil {
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}
		h.logger.Error("Failed to delete proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "代理删除成功", nil)
}

// TestProxy 测试代理
func (h *ProxyHandler) TestProxy(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的代理ID")
		return
	}

	result, err := h.proxyService.TestProxy(userID, proxyID)
	if err != nil {
		if err == services.ErrProxyNotFound {
			response.ProxyNotFound(c)
			return
		}
		h.logger.Error("Failed to test proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "代理测试完成", result)
}

// GetProxyStats 获取代理统计
func (h *ProxyHandler) GetProxyStats(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	stats, err := h.proxyService.GetProxyStats(userID)
	if err != nil {
		h.logger.Error("Failed to get proxy stats",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取代理统计失败")
		return
	}

	response.Success(c, stats)
}
