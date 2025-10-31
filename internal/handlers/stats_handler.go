package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/services"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	statsService services.StatsService
	logger       *zap.Logger
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(statsService services.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
		logger:       logger.Get().Named("stats_handler"),
	}
}

// GetOverview 获取系统统计概览
// @Summary 获取系统统计概览
// @Description 获取系统整体运行统计数据，包括用户、账号、任务等核心指标
// @Tags 统计
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param period query string false "统计周期" Enums(day, week, month) default(week)
// @Success 200 {object} models.SystemOverview "系统统计概览"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/stats/overview [get]
func (h *StatsHandler) GetOverview(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 获取统计周期参数
	period := c.DefaultQuery("period", "week")

	overview, err := h.statsService.GetSystemOverview(c.Request.Context(), userID, period)
	if err != nil {
		h.logger.Error("Failed to get system overview",
			zap.Uint64("user_id", userID),
			zap.String("period", period),
			zap.Error(err))
		response.InternalError(c, "获取统计数据失败")
		return
	}

	response.Success(c, overview)
}

// GetAccountStats 获取账号统计
// @Summary 获取账号统计详情
// @Description 获取用户账号的详细统计信息，包括状态分布、健康度等
// @Tags 统计
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param period query string false "统计周期" Enums(day, week, month) default(week)
// @Param status query string false "账号状态过滤"
// @Success 200 {object} models.AccountStatistics "账号统计详情"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/stats/accounts [get]
func (h *StatsHandler) GetAccountStats(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 获取查询参数
	period := c.DefaultQuery("period", "week")
	status := c.Query("status")

	accountStats, err := h.statsService.GetAccountStatistics(c.Request.Context(), userID, period, status)
	if err != nil {
		h.logger.Error("Failed to get account statistics",
			zap.Uint64("user_id", userID),
			zap.String("period", period),
			zap.String("status", status),
			zap.Error(err))
		response.InternalError(c, "获取账号统计失败")
		return
	}

	response.Success(c, accountStats)
}

// GetUserDashboard 获取用户仪表盘数据
// @Summary 获取用户仪表盘
// @Description 获取用户个人仪表盘的核心数据
// @Tags 统计
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.UserDashboard "用户仪表盘数据"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/stats/dashboard [get]
func (h *StatsHandler) GetUserDashboard(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	dashboard, err := h.statsService.GetUserDashboard(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user dashboard",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取仪表盘数据失败")
		return
	}

	response.Success(c, dashboard)
}
