-- 移除代理Host字段的迁移脚本
-- 执行时间: 2024-XX-XX

-- 从 proxy_ips 表中移除 host 字段
ALTER TABLE proxy_ips DROP COLUMN IF EXISTS host;

-- 说明：
-- Host字段已被移除，代理配置只需要IP地址即可
-- 如果之前有数据依赖Host字段，请确保已迁移到IP字段
