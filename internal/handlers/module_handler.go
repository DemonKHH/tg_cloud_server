package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// ModuleHandler 模块功能处理器
type ModuleHandler struct {
	taskService    *services.TaskService
	accountService *services.AccountService
	logger         *zap.Logger
}

// NewModuleHandler 创建模块处理器
func NewModuleHandler(taskService *services.TaskService, accountService *services.AccountService) *ModuleHandler {
	return &ModuleHandler{
		taskService:    taskService,
		accountService: accountService,
		logger:         logger.Get().Named("module_handler"),
	}
}

// AccountCheck 账号检查模块
// @Summary 执行账号检查
// @Description 对指定账号执行健康度检查任务
// @Tags 模块功能
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body ModuleTaskRequest true "任务请求，必须包含account_id"
// @Success 201 {object} models.Task "创建的任务"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 422 {object} map[string]string "账号验证失败"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/modules/check [post]
func (h *ModuleHandler) AccountCheck(c *gin.Context) {
	task, err := h.createModuleTask(c, models.TaskTypeCheck, map[string]interface{}{
		"check_type": "health_check",
		"timeout":    "2m",
	})
	if err != nil {
		return // 错误已在createModuleTask中处理
	}

	h.logger.Info("Account check task created",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", task.GetAccountIDList()))

	response.SuccessWithMessage(c, "账号检查任务创建成功", task)
}

// PrivateMessage 私信模块
// @Summary 发送私信
// @Description 通过指定账号发送私信给目标用户
// @Tags 模块功能
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body PrivateMessageRequest true "私信请求，必须包含account_id"
// @Success 201 {object} models.Task "创建的任务"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 422 {object} map[string]string "账号验证失败"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/modules/private [post]
func (h *ModuleHandler) PrivateMessage(c *gin.Context) {
	var req PrivateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid private message request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 验证必需的配置参数
	if len(req.Targets) == 0 {
		response.InvalidParam(c, "缺少目标用户列表")
		return
	}

	if req.Message == "" {
		response.InvalidParam(c, "缺少消息内容")
		return
	}

	taskConfig := map[string]interface{}{
		"targets": req.Targets,
		"message": req.Message,
	}

	if req.DelayBetween > 0 {
		taskConfig["delay_between"] = req.DelayBetween
	}

	task, err := h.createModuleTask(c, models.TaskTypePrivate, taskConfig)
	if err != nil {
		return
	}

	h.logger.Info("Private message task created",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", task.GetAccountIDList()),
		zap.Int("target_count", len(req.Targets)))

	response.SuccessWithMessage(c, "私信任务创建成功", task)
}

// Broadcast 群发模块
// @Summary 群发消息
// @Description 通过指定账号向群组或频道群发消息
// @Tags 模块功能
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body BroadcastRequest true "群发请求，必须包含account_id"
// @Success 201 {object} models.Task "创建的任务"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 422 {object} map[string]string "账号验证失败"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/modules/broadcast [post]
func (h *ModuleHandler) Broadcast(c *gin.Context) {
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid broadcast request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	// 验证必需的配置参数
	if len(req.Groups) == 0 && len(req.Channels) == 0 {
		response.InvalidParam(c, "缺少目标群组或频道")
		return
	}

	if req.Message == "" {
		response.InvalidParam(c, "缺少消息内容")
		return
	}

	taskConfig := map[string]interface{}{
		"message": req.Message,
	}

	if len(req.Groups) > 0 {
		taskConfig["groups"] = req.Groups
	}
	if len(req.Channels) > 0 {
		taskConfig["channels"] = req.Channels
	}
	if req.DelayBetween > 0 {
		taskConfig["delay_between"] = req.DelayBetween
	}

	task, err := h.createModuleTask(c, models.TaskTypeBroadcast, taskConfig)
	if err != nil {
		return
	}

	totalTargets := len(req.Groups) + len(req.Channels)
	h.logger.Info("Broadcast task created",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", task.GetAccountIDList()),
		zap.Int("total_targets", totalTargets))

	response.SuccessWithMessage(c, "群发任务创建成功", task)
}

// VerifyCode 验证码接收模块 (已弃用，建议使用新的验证码API)
// @Summary 接收验证码 (弃用)
// @Description 此接口已弃用，建议使用 /api/v1/verify-code/generate 接口生成访问链接
// @Tags 模块功能
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body VerifyCodeRequest true "验证码请求，必须包含account_id"
// @Success 200 {object} map[string]interface{} "弃用提示和迁移指南"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Router /api/v1/modules/verify [post]
func (h *ModuleHandler) VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid verify code request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	h.logger.Warn("Using deprecated verify code API",
		zap.Uint64("account_id", req.AccountID),
		zap.String("suggestion", "Please use /api/v1/verify-code/generate instead"))

	// 返回弃用警告和新API建议
	response.Success(c, gin.H{
		"deprecated": true,
		"message":    "此接口已弃用，请使用新的验证码API",
		"new_api": gin.H{
			"generate_link": "/api/v1/verify-code/generate",
			"description":   "先调用generate获取访问链接，然后访问链接获取验证码",
		},
		"migration_guide": gin.H{
			"step1": fmt.Sprintf("POST /api/v1/verify-code/generate with {\"account_id\": %d}", req.AccountID),
			"step2": "GET /api/v1/verify-code/{code} from the returned URL",
		},
	})
}

// GroupChat AI炒群模块
// @Summary AI炒群
// @Description 使用指定账号在群组中进行AI智能互动
// @Tags 模块功能
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body GroupChatRequest true "AI炒群请求，必须包含account_id"
// @Success 201 {object} models.Task "创建的任务"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 422 {object} map[string]string "账号验证失败"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/modules/groupchat [post]
func (h *ModuleHandler) GroupChat(c *gin.Context) {
	var req GroupChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid group chat request", zap.Error(err))
		response.InvalidParam(c, "请求参数无效："+err.Error())
		return
	}

	if req.GroupID == 0 {
		response.InvalidParam(c, "缺少群组ID")
		return
	}

	taskConfig := map[string]interface{}{
		"group_id":  req.GroupID,
		"duration":  req.Duration,
		"ai_config": req.AIConfig,
	}

	task, err := h.createModuleTask(c, models.TaskTypeGroupChat, taskConfig)
	if err != nil {
		return
	}

	h.logger.Info("Group chat task created",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", task.GetAccountIDList()),
		zap.Int64("group_id", req.GroupID))

	response.SuccessWithMessage(c, "AI炒群任务创建成功", task)
}

// createModuleTask 创建模块任务的通用方法
func (h *ModuleHandler) createModuleTask(c *gin.Context, taskType models.TaskType, taskConfig map[string]interface{}) (*models.Task, error) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未找到用户信息")
		return nil, gin.Error{}
	}

	uid, ok := userID.(uint64)
	if !ok {
		response.Unauthorized(c, "用户ID格式错误")
		return nil, gin.Error{}
	}

	// 获取account_id（所有模块都统一使用这个参数名）
	var accountID uint64
	if err := c.ShouldBindJSON(&struct {
		AccountID uint64 `json:"account_id" binding:"required"`
	}{
		AccountID: accountID,
	}); err != nil {
		response.InvalidParam(c, "所有任务都必须指定account_id参数")
		return nil, gin.Error{}
	}

	// 验证账号可用性
	validation, err := h.accountService.ValidateAccountForTask(uid, accountID, taskType)
	if err != nil {
		h.logger.Error("Failed to validate account",
			zap.Uint64("user_id", uid),
			zap.Uint64("account_id", accountID),
			zap.String("task_type", string(taskType)),
			zap.Error(err))
		response.InternalError(c, "账号验证失败")
		return nil, gin.Error{}
	}

	if !validation.IsValid {
		response.ErrorWithData(c, response.CodeInternalError, "账号不可用", gin.H{
			"warnings": validation.Warnings,
			"errors":   validation.Errors,
		})
		return nil, gin.Error{}
	}

	// 创建任务请求
	createReq := &models.CreateTaskRequest{
		AccountIDs: []uint64{accountID},
		TaskType:   taskType,
		Config:     taskConfig,
		Priority:   5, // 默认优先级
	}

	// 创建任务
	task, err := h.taskService.CreateTask(uid, createReq)
	if err != nil {
		h.logger.Error("Failed to create task",
			zap.Uint64("user_id", uid),
			zap.Uint64("account_id", accountID),
			zap.String("task_type", string(taskType)),
			zap.Error(err))
		response.InternalError(c, "任务创建失败")
		return nil, gin.Error{}
	}

	return task, nil
}

// 请求结构体定义

// ModuleTaskRequest 基础模块任务请求
type ModuleTaskRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"` // 统一使用account_id
}

// PrivateMessageRequest 私信请求
type PrivateMessageRequest struct {
	AccountID    uint64   `json:"account_id" binding:"required"`
	Targets      []string `json:"targets" binding:"required"`
	Message      string   `json:"message" binding:"required"`
	DelayBetween int      `json:"delay_between,omitempty"` // 发送间隔(秒)
}

// BroadcastRequest 群发请求
type BroadcastRequest struct {
	AccountID    uint64  `json:"account_id" binding:"required"`
	Groups       []int64 `json:"groups,omitempty"`   // 目标群组ID列表
	Channels     []int64 `json:"channels,omitempty"` // 目标频道ID列表
	Message      string  `json:"message" binding:"required"`
	DelayBetween int     `json:"delay_between,omitempty"` // 发送间隔(秒)
}

// VerifyCodeRequest 验证码请求
type VerifyCodeRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"`
	Timeout   int    `json:"timeout,omitempty"` // 超时时间(秒)，默认30秒
	Source    string `json:"source,omitempty"`  // 验证码来源过滤
	Pattern   string `json:"pattern,omitempty"` // 验证码匹配模式
}

// GroupChatRequest AI炒群请求
type GroupChatRequest struct {
	AccountID uint64                 `json:"account_id" binding:"required"`
	GroupID   int64                  `json:"group_id" binding:"required"`
	Duration  int                    `json:"duration,omitempty"`  // 持续时间(分钟)
	AIConfig  map[string]interface{} `json:"ai_config,omitempty"` // AI配置
}
