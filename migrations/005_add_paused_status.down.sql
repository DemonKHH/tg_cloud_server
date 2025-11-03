-- 移除索引
ALTER TABLE `tasks` 
DROP INDEX `idx_tasks_status_priority`,
DROP INDEX `idx_tasks_user_status_priority`;

-- 移除paused状态（注意：这会丢失处于paused状态的任务数据）
ALTER TABLE `tasks` 
MODIFY COLUMN `status` ENUM('pending', 'queued', 'running', 'completed', 'failed', 'cancelled') 
NOT NULL DEFAULT 'pending' COMMENT '任务状态';
