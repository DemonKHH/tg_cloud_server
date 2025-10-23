package models

// 补充模型定义，用于统一仓库接口

// Proxy 代理模型别名（用于仓库接口统一）
type Proxy = ProxyIP

// ProxyStats 代理统计信息（仓库接口版本）
type ProxyStats struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	Error    int64 `json:"error"`
	Testing  int64 `json:"testing"`
}

// TaskStats 任务统计信息（仓库接口版本）
type TaskStats struct {
	Total      int64 `json:"total"`
	Pending    int64 `json:"pending"`
	Running    int64 `json:"running"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Cancelled  int64 `json:"cancelled"`
	TodayTasks int64 `json:"today_tasks"`
}

// QueueInfo 队列信息（仓库接口版本）
type QueueInfo struct {
	AccountID         uint64 `json:"account_id"`
	PendingTasks      int64  `json:"pending_tasks"`
	RunningTasks      int64  `json:"running_tasks"`
	EstimatedWaitTime int64  `json:"estimated_wait_time"` // 秒
}
