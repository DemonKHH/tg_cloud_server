package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// ProxyHandler 代理处理器
type ProxyHandler struct {
	accountService *services.AccountService
	logger         *zap.Logger
}

// NewProxyHandler 创建代理处理器
func NewProxyHandler(accountService *services.AccountService) *ProxyHandler {
	return &ProxyHandler{
		accountService: accountService,
		logger:         logger.Get().Named("proxy_handler"),
	}
}

// CreateProxy 创建代理
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	userID := getUserID(c)

	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxy, err := h.accountService.CreateProxy(userID, &req)
	if err != nil {
		h.logger.Error("Failed to create proxy",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Proxy created successfully",
		"data":    proxy,
	})
}

// GetProxies 获取代理列表
func (h *ProxyHandler) GetProxies(c *gin.Context) {
	userID := getUserID(c)

	status := c.Query("status")
	var proxies []*models.ProxyIP
	var err error

	if status != "" {
		proxies, err = h.accountService.GetProxiesByStatus(userID, status)
	} else {
		proxies, err = h.accountService.GetProxies(userID)
	}

	if err != nil {
		h.logger.Error("Failed to get proxies",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get proxies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": proxies,
	})
}

// GetProxy 获取代理详情
func (h *ProxyHandler) GetProxy(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proxy ID"})
		return
	}

	proxy, err := h.accountService.GetProxy(userID, proxyID)
	if err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Proxy not found"})
			return
		}
		h.logger.Error("Failed to get proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get proxy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": proxy,
	})
}

// UpdateProxy 更新代理
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proxy ID"})
		return
	}

	var req models.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxy, err := h.accountService.UpdateProxy(userID, proxyID, &req)
	if err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Proxy not found"})
			return
		}
		h.logger.Error("Failed to update proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Proxy updated successfully",
		"data":    proxy,
	})
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proxy ID"})
		return
	}

	if err := h.accountService.DeleteProxy(userID, proxyID); err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Proxy not found"})
			return
		}
		h.logger.Error("Failed to delete proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Proxy deleted successfully",
	})
}

// TestProxy 测试代理
func (h *ProxyHandler) TestProxy(c *gin.Context) {
	userID := getUserID(c)
	proxyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proxy ID"})
		return
	}

	result, err := h.accountService.TestProxy(userID, proxyID)
	if err != nil {
		if err == services.ErrProxyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Proxy not found"})
			return
		}
		h.logger.Error("Failed to test proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Proxy test completed",
		"data":    result,
	})
}

// GetProxyStats 获取代理统计
func (h *ProxyHandler) GetProxyStats(c *gin.Context) {
	userID := getUserID(c)

	stats, err := h.accountService.GetProxyStats(userID)
	if err != nil {
		h.logger.Error("Failed to get proxy stats",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get proxy stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}
