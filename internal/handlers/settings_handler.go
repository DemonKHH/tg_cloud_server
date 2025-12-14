package handlers

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// SettingsHandler 设置处理器
type SettingsHandler struct {
	riskControlService services.RiskControlService
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(riskControlService services.RiskControlService) *SettingsHandler {
	return &SettingsHandler{
		riskControlService: riskControlService,
	}
}

// GetRiskSettings 获取风控配置
// @Summary 获取风控配置
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} models.UserRiskSettings
// @Router /api/v1/settings/risk [get]
func (h *SettingsHandler) GetRiskSettings(c *gin.Context) {
	userID := c.GetUint64("user_id")

	settings := h.riskControlService.GetUserRiskSettings(c.Request.Context(), userID)

	response.Success(c, settings)
}

// UpdateRiskSettings 更新风控配置
// @Summary 更新风控配置
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.UpdateRiskSettingsRequest true "风控配置"
// @Success 200 {object} models.UserRiskSettings
// @Router /api/v1/settings/risk [put]
func (h *SettingsHandler) UpdateRiskSettings(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req models.UpdateRiskSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, "参数错误: "+err.Error())
		return
	}

	settings := &models.UserRiskSettings{
		MaxConsecutiveFailures: req.MaxConsecutiveFailures,
		CoolingDurationMinutes: req.CoolingDurationMinutes,
	}

	if err := h.riskControlService.UpdateUserRiskSettings(c.Request.Context(), userID, settings); err != nil {
		response.InternalError(c, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", settings)
}
