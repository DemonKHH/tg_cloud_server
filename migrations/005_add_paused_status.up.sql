-- 添加paused状态到任务状态枚举
ALTER TABLE `tasks` 
MODIFY COLUMN `status` ENUM('pending', 'queued', 'running', 'paused', 'completed', 'failed', 'cancelled') 
NOT NULL DEFAULT 'pending' COMMENT '任务状态';

-- 添加索引优化
ALTER TABLE `tasks` 
ADD INDEX `idx_tasks_status_priority` (`status`, `priority`),
ADD INDEX `idx_tasks_user_status_priority` (`user_id`, `status`, `priority`);
