-- 创建风控日志表
CREATE TABLE IF NOT EXISTS `risk_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账号ID',
    `risk_type` VARCHAR(50) NOT NULL COMMENT '风险类型',
    `risk_level` ENUM('low', 'medium', 'high', 'critical') NOT NULL COMMENT '风险级别',
    `description` TEXT COMMENT '风险描述',
    `action_taken` VARCHAR(100) COMMENT '采取的行动',
    `metadata` JSON COMMENT '元数据(JSON格式)',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_risk_logs_account_id` (`account_id`),
    INDEX `idx_risk_logs_risk_type` (`risk_type`),
    INDEX `idx_risk_logs_risk_level` (`risk_level`),
    INDEX `idx_risk_logs_created_at` (`created_at`),
    INDEX `idx_risk_logs_account_risk_type` (`account_id`, `risk_type`),
    FOREIGN KEY (`account_id`) REFERENCES `tg_accounts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='风控日志表';
