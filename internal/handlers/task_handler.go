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
	userID := getUserID(c)

	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.CreateTask(userID, &req)
	if err != nil {
		h.logger.Error("Failed to create task",
			zap.Uint64("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"data":    task,
	})
}

// GetTasks 获取任务列表
func (h *TaskHandler) GetTasks(c *gin.Context) {
	userID := getUserID(c)

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tasks": tasks,
			"pagination": gin.H{
				"current_page": filter.Page,
				"per_page":     filter.Limit,
				"total":        total,
				"total_pages":  (total + int64(filter.Limit) - 1) / int64(filter.Limit),
			},
		},
	})
}

// GetTask 获取任务详情
func (h *TaskHandler) GetTask(c *gin.Context) {
	userID := getUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.taskService.GetTask(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.Error("Failed to get task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": task,
	})
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID := getUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req services.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.UpdateTask(userID, taskID, &req)
	if err != nil {
		if err == services.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.Error("Failed to update task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task updated successfully",
		"data":    task,
	})
}

// CancelTask 取消任务
func (h *TaskHandler) CancelTask(c *gin.Context) {
	userID := getUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	if err := h.taskService.CancelTask(userID, taskID); err != nil {
		if err == services.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.Error("Failed to cancel task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled successfully",
	})
}

// RetryTask 重试任务
func (h *TaskHandler) RetryTask(c *gin.Context) {
	userID := getUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.taskService.RetryTask(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.Error("Failed to retry task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task retry scheduled",
		"data":    task,
	})
}

// GetTaskLogs 获取任务日志
func (h *TaskHandler) GetTaskLogs(c *gin.Context) {
	userID := getUserID(c)
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	logs, err := h.taskService.GetTaskLogs(userID, taskID)
	if err != nil {
		if err == services.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.Error("Failed to get task logs",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
	})
}

// GetTaskStats 获取任务统计
func (h *TaskHandler) GetTaskStats(c *gin.Context) {
	userID := getUserID(c)
	timeRange := c.DefaultQuery("range", "all")

	stats, err := h.taskService.GetTaskStats(userID, timeRange)
	if err != nil {
		h.logger.Error("Failed to get task stats",
			zap.Uint64("user_id", userID),
			zap.String("range", timeRange),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// BatchCancel 批量取消任务
func (h *TaskHandler) BatchCancel(c *gin.Context) {
	userID := getUserID(c)

	var req services.BatchCancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	successCount, err := h.taskService.BatchCancelTasks(userID, req.TaskIDs)
	if err != nil {
		h.logger.Error("Failed to batch cancel tasks",
			zap.Uint64("user_id", userID),
			zap.Int("total", len(req.TaskIDs)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to batch cancel tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch cancel completed",
		"data": gin.H{
			"total":   len(req.TaskIDs),
			"success": successCount,
			"failed":  len(req.TaskIDs) - successCount,
		},
	})
}

// GetQueueInfo 获取队列信息
func (h *TaskHandler) GetQueueInfo(c *gin.Context) {
	userID := getUserID(c)
	accountID, err := strconv.ParseUint(c.Param("account_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	info, err := h.taskService.GetQueueInfo(userID, accountID)
	if err != nil {
		h.logger.Error("Failed to get queue info",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": info,
	})
}

// CleanupTasks 清理已完成任务
func (h *TaskHandler) CleanupTasks(c *gin.Context) {
	userID := getUserID(c)

	var req services.CleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count, err := h.taskService.CleanupCompletedTasks(userID, req.OlderThanDays)
	if err != nil {
		h.logger.Error("Failed to cleanup tasks",
			zap.Uint64("user_id", userID),
			zap.Int("days", req.OlderThanDays),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tasks cleaned up successfully",
		"data": gin.H{
			"deleted_count": count,
		},
	})
}
