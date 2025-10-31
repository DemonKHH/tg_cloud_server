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

// TaskHandler 任务处理器
type TaskHandler struct {
	taskService *services.TaskService
	logger      *zap.Logger
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger.Get().Named("task_handler"),
	}
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	task, err := h.taskService.CreateTask(userID, &req)
	if err != nil {
		h.logger.Error("Failed to create task",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "任务创建成功", task)
}

// GetTasks 获取任务列表
func (h *TaskHandler) GetTasks(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 解析查询参数
	filter := &services.TaskFilter{
		UserID: userID,
		Page:   1,
		Limit:  20,
	}

	if accountID := c.Query("account_id"); accountID != "" {
		if id, err := strconv.ParseUint(accountID, 10, 64); err == nil {
			filter.AccountID = id
		}
	}

	if taskType := c.Query("task_type"); taskType != "" {
		filter.TaskType = taskType
	}

	if status := c.Query("status"); status != "" {
		filter.Status = status
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	tasks, total, err := h.taskService.GetTasks(filter)
	if err != nil {
		h.logger.Error("Failed to get tasks",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		response.InternalError(c, "获取任务列表失败")
		return
	}

	response.Paginated(c, tasks, filter.Page, filter.Limit, total)
}

// GetTask 获取任务详情
func (h *TaskHandler) GetTask(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的任务ID")
		return
	}

	task, err := h.taskService.GetTask(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			response.TaskNotFound(c)
			return
		}
		h.logger.Error("Failed to get task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		response.InternalError(c, "获取任务失败")
		return
	}

	response.Success(c, task)
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的任务ID")
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	task, err := h.taskService.UpdateTask(userID, taskID, &req)
	if err != nil {
		if err == services.ErrTaskNotFound {
			response.TaskNotFound(c)
			return
		}
		h.logger.Error("Failed to update task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "任务更新成功", task)
}

// CancelTask 取消任务
func (h *TaskHandler) CancelTask(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的任务ID")
		return
	}

	if err := h.taskService.CancelTask(userID, taskID); err != nil {
		if err == services.ErrTaskNotFound {
			response.TaskNotFound(c)
			return
		}
		h.logger.Error("Failed to cancel task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "任务取消成功", nil)
}

// RetryTask 重试任务
func (h *TaskHandler) RetryTask(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的任务ID")
		return
	}

	task, err := h.taskService.RetryTask(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			response.TaskNotFound(c)
			return
		}
		h.logger.Error("Failed to retry task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "任务重试已调度", task)
}

// GetTaskLogs 获取任务日志
func (h *TaskHandler) GetTaskLogs(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的任务ID")
		return
	}

	logs, err := h.taskService.GetTaskLogs(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			response.TaskNotFound(c)
			return
		}
		h.logger.Error("Failed to get task logs",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		response.InternalError(c, "获取任务日志失败")
		return
	}

	response.Success(c, logs)
}

// GetTaskStats 获取任务统计
func (h *TaskHandler) GetTaskStats(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	timeRange := c.DefaultQuery("range", "all")

	stats, err := h.taskService.GetTaskStats(userID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get task stats",
			zap.Uint64("user_id", userID),
			zap.String("range", timeRange),
			zap.Error(err))
		response.InternalError(c, "获取任务统计失败")
		return
	}

	response.Success(c, stats)
}

// BatchCancel 批量取消任务
func (h *TaskHandler) BatchCancel(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.BatchCancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	successCount, err := h.taskService.BatchCancelTasks(userID, req.TaskIDs)
	if err != nil {
		h.logger.Error("Failed to batch cancel tasks",
			zap.Uint64("user_id", userID),
			zap.Int("total", len(req.TaskIDs)),
			zap.Error(err))
		response.InternalError(c, "批量取消任务失败")
		return
	}

	response.SuccessWithMessage(c, "批量取消完成", gin.H{
		"total":   len(req.TaskIDs),
		"success": successCount,
		"failed":  len(req.TaskIDs) - successCount,
	})
}

// GetQueueInfo 获取队列信息
func (h *TaskHandler) GetQueueInfo(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	accountID, err := strconv.ParseUint(c.Param("account_id"), 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的账号ID")
		return
	}

	info, err := h.taskService.GetQueueInfo(userID, accountID)
	if err != nil {
		h.logger.Error("Failed to get queue info",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, info)
}

// CleanupTasks 清理已完成任务
func (h *TaskHandler) CleanupTasks(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.CleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	count, err := h.taskService.CleanupCompletedTasks(userID, req.OlderThanDays)
	if err != nil {
		h.logger.Error("Failed to cleanup tasks",
			zap.Uint64("user_id", userID),
			zap.Int("days", req.OlderThanDays),
			zap.Error(err))
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "任务清理成功", gin.H{
		"deleted_count": count,
	})
}
