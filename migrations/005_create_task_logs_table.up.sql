-- 创建任务日志表
CREATE TABLE IF NOT EXISTS `task_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `task_id` BIGINT UNSIGNED NOT NULL COMMENT '任务ID',
    `account_id` BIGINT UNSIGNED NULL COMMENT '账号ID',
    `action` VARCHAR(50) NOT NULL COMMENT '操作类型',
    `message` TEXT COMMENT '日志消息',
    `extra_data` JSON COMMENT '额外数据(JSON格式)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_task_logs_task_id` (`task_id`),
    INDEX `idx_task_logs_account_id` (`account_id`),
    INDEX `idx_task_logs_action` (`action`),
    INDEX `idx_task_logs_created_at` (`created_at`),
    FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON DELETE CASCADE,
    FOREIGN KEY (`account_id`) REFERENCES `tg_accounts` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务日志表';
