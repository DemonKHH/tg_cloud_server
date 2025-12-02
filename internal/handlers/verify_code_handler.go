package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// VerifyCodeHandler 验证码处理器
type VerifyCodeHandler struct {
	verifyCodeService *services.VerifyCodeService
	logger            *zap.Logger
}

// NewVerifyCodeHandler 创建验证码处理器
func NewVerifyCodeHandler(verifyCodeService *services.VerifyCodeService) *VerifyCodeHandler {
	return &VerifyCodeHandler{
		verifyCodeService: verifyCodeService,
		logger:            zap.L().Named("verify_code_handler"),
	}
}

// GenerateCode 生成验证码访问链接
// @Summary 生成验证码访问链接
// @Description 为指定TG账号生成临时的验证码访问链接
// @Tags 验证码
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.GenerateCodeRequest true "生成验证码链接请求"
// @Success 201 {object} models.GenerateCodeResponse "生成的访问链接信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "账号不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/verify-code/generate [post]
func (h *VerifyCodeHandler) GenerateCode(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.GenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid generate code request",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 生成临时访问代码
	codeResponse, err := h.verifyCodeService.GenerateCode(userID, req.AccountID, req.ExpiresIn)
	if err != nil {
		h.logger.Error("Failed to generate verification code",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", req.AccountID),
			zap.Error(err))

		// 根据错误类型返回相应响应
		if verifyErr, ok := err.(*models.VerifyCodeError); ok {
			switch verifyErr.Code {
			case "ACCOUNT_NOT_FOUND":
				response.NotFound(c, verifyErr.Message)
			default:
				response.InternalError(c, verifyErr.Message)
			}
		} else {
			response.InternalError(c, "生成验证码访问链接失败")
		}
		return
	}

	h.logger.Info("Verification code link generated successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", req.AccountID),
		zap.String("code", codeResponse.Code),
		zap.Int("expires_in", codeResponse.ExpiresIn))

	response.SuccessWithMessage(c, "验证码访问链接生成成功", codeResponse)
	response.SuccessWithMessage(c, "验证码访问链接生成成功", codeResponse)
}

// BatchGenerateCode 批量生成验证码访问链接
// @Summary 批量生成验证码访问链接
// @Description 为多个TG账号生成临时的验证码访问链接
// @Tags 验证码
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.BatchGenerateCodeRequest true "批量生成验证码链接请求"
// @Success 201 {object} map[string]interface{} "生成的访问链接信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/verify-code/batch/generate [post]
func (h *VerifyCodeHandler) BatchGenerateCode(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.BatchGenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid batch generate code request",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 批量生成临时访问代码
	results, err := h.verifyCodeService.BatchGenerateCode(userID, req.AccountIDs, req.ExpiresIn)
	if err != nil {
		h.logger.Error("Failed to batch generate verification codes",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "批量生成验证码访问链接失败")
		return
	}

	h.logger.Info("Batch verification code links generated successfully",
		zap.Uint64("user_id", userID),
		zap.Int("count", len(results)))

	response.SuccessWithMessage(c, "批量生成成功", gin.H{
		"items": results,
	})
}

// ListSessions 获取验证码会话列表
// @Summary 获取验证码会话列表
// @Description 获取当前用户的所有活跃验证码会话 (分页)
// @Tags 验证码
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(50)
// @Success 200 {object} map[string]interface{} "会话列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/verify-code/sessions [get]
func (h *VerifyCodeHandler) ListSessions(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 获取分页参数
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	sessions, total, err := h.verifyCodeService.ListSessions(userID, page, limit)
	if err != nil {
		h.logger.Error("Failed to list verification code sessions",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取会话列表失败")
		return
	}

	response.Paginated(c, sessions, page, limit, total)
}

// GetVerifyCode 通过访问码获取验证码 (公开接口，不需要认证)
// @Summary 获取验证码
// @Description 通过访问码直接获取TG验证码，此接口不需要认证
// @Tags 验证码
// @Produce json
// @Param code path string true "临时访问代码"
// @Param timeout query int false "超时时间(秒)，默认60秒，最大300秒"
// @Success 200 {object} models.VerifyCodeResponse "验证码信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 404 {object} map[string]string "访问码无效"
// @Failure 408 {object} models.VerifyCodeResponse "验证码接收超时"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/verify-code/{code} [get]
func (h *VerifyCodeHandler) GetVerifyCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.InvalidParam(c, "访问代码不能为空")
		return
	}

	// 获取超时参数
	timeoutSeconds := 60 // 默认60秒
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			timeoutSeconds = timeout
		}
	}

	h.logger.Info("Starting verification code retrieval",
		zap.String("code", code),
		zap.Int("timeout_seconds", timeoutSeconds),
		zap.String("client_ip", c.ClientIP()))

	// 获取验证码
	verifyResult, err := h.verifyCodeService.GetVerifyCode(c.Request.Context(), code, timeoutSeconds)
	if err != nil {
		h.logger.Warn("Verification code retrieval failed",
			zap.String("code", code),
			zap.Error(err))

		// 根据错误类型返回相应响应
		if verifyErr, ok := err.(*models.VerifyCodeError); ok {
			switch verifyErr.Code {
			case "CODE_NOT_FOUND", "CODE_EXPIRED":
				response.NotFound(c, verifyErr.Message)
			case "VERIFY_TIMEOUT":
				c.JSON(408, gin.H{
					"success": false,
					"message": verifyErr.Message,
					"data":    verifyResult,
				})
			case "ACCOUNT_NOT_FOUND", "TELEGRAM_CONNECTION_ERROR":
				response.InternalError(c, verifyErr.Message)
			default:
				response.InternalError(c, verifyErr.Message)
			}
		} else {
			response.InternalError(c, "验证码获取失败")
		}
		return
	}

	if verifyResult.Success {
		h.logger.Info("Verification code retrieved successfully",
			zap.String("code", code),
			zap.String("verify_code", verifyResult.Code),
			zap.String("sender", verifyResult.Sender),
			zap.Int("wait_seconds", verifyResult.WaitSeconds))

		response.SuccessWithMessage(c, "验证码获取成功", verifyResult)
	} else {
		h.logger.Warn("Verification code retrieval timeout",
			zap.String("code", code),
			zap.Int("wait_seconds", verifyResult.WaitSeconds))

		c.JSON(408, gin.H{
			"success": false,
			"message": verifyResult.Message,
			"data":    verifyResult,
		})
	}
}

// GetCodeInfo 获取访问码信息 (用于调试，需要认证)
// @Summary 获取访问码信息
// @Description 获取访问码的详细信息，用于调试
// @Tags 验证码
// @Produce json
// @Security ApiKeyAuth
// @Param code path string true "临时访问代码"
// @Success 200 {object} models.VerifyCodeSession "访问码会话信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "访问码不存在"
// @Router /api/v1/verify-code/{code}/info [get]
func (h *VerifyCodeHandler) GetCodeInfo(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	code := c.Param("code")
	if code == "" {
		response.InvalidParam(c, "访问代码不能为空")
		return
	}

	// 获取会话信息
	session := h.verifyCodeService.GetSessionInfo(code)
	if session == nil {
		response.NotFound(c, "访问码不存在或已过期")
		return
	}

	// 只允许访问自己的会话信息
	if session.UserID != userID {
		response.Forbidden(c, "无权限访问此访问码信息")
		return
	}

	h.logger.Info("Code info retrieved",
		zap.String("code", code),
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", session.AccountID))

	response.Success(c, session)
}
