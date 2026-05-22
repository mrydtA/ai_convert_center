-- TokenHub MySQL 初始化脚本
-- 此脚本会在 MySQL 容器首次启动时自动执行

-- 创建扩展表（如果需要）
USE tokenhub;

-- 充值记录表（新开发功能）
CREATE TABLE IF NOT EXISTS `topups` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `user_id` INT NOT NULL,
    `order_no` VARCHAR(64) UNIQUE NOT NULL,
    `amount` DECIMAL(10,2) NOT NULL,
    `payment_method` ENUM('alipay', 'wechat', 'payspi', 'manual') NOT NULL,
    `status` ENUM('pending', 'completed', 'failed', 'refunded') DEFAULT 'pending',
    `transaction_id` VARCHAR(128),
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_order_no` (`order_no`),
    INDEX `idx_status` (`status`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='充值记录表';

-- 套餐表（新开发功能）
CREATE TABLE IF NOT EXISTS `plans` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(128) NOT NULL,
    `description` TEXT,
    `price` DECIMAL(10,2) NOT NULL,
    `credits` DECIMAL(10,2) NOT NULL,
    `duration_days` INT DEFAULT 30,
    `status` TINYINT DEFAULT 1 COMMENT '1=上架，0=下架',
    `sort_order` INT DEFAULT 0,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_status` (`status`),
    INDEX `idx_sort` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='套餐表';

-- 支付流水表（新开发功能）
CREATE TABLE IF NOT EXISTS `payment_transactions` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` INT NOT NULL,
    `order_no` VARCHAR(64) NOT NULL,
    `transaction_id` VARCHAR(128),
    `payment_method` ENUM('alipay', 'wechat', 'payspi') NOT NULL,
    `amount` DECIMAL(10,2) NOT NULL,
    `currency` VARCHAR(10) DEFAULT 'CNY',
    `status` ENUM('pending', 'success', 'failed', 'refunded') DEFAULT 'pending',
    `request_data` JSON,
    `response_data` JSON,
    `callback_time` TIMESTAMP NULL,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_order_no` (`order_no`),
    INDEX `idx_transaction_id` (`transaction_id`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付流水表';

-- 渠道统计表（新开发功能）
CREATE TABLE IF NOT EXISTS `channel_stats` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `channel_id` INT NOT NULL,
    `stat_date` DATE NOT NULL,
    `total_requests` BIGINT DEFAULT 0,
    `total_tokens` BIGINT DEFAULT 0,
    `total_cost` DECIMAL(10,4) DEFAULT 0,
    `success_count` BIGINT DEFAULT 0,
    `failure_count` BIGINT DEFAULT 0,
    `avg_latency_ms` INT DEFAULT 0,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_channel_date` (`channel_id`, `stat_date`),
    INDEX `idx_stat_date` (`stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='渠道统计表';

-- 插入示例套餐数据
INSERT INTO `plans` (`name`, `description`, `price`, `credits`, `duration_days`, `status`, `sort_order`) VALUES
('体验套餐', '适合初次用户体验', 9.90, 10.00, 7, 1, 1),
('基础套餐', '适合个人开发者', 49.00, 55.00, 30, 1, 2),
('专业套餐', '适合小型团队', 199.00, 240.00, 30, 1, 3),
('企业套餐', '适合企业用户', 999.00, 1300.00, 30, 1, 4);

-- 用户表扩展字段（如果 New API 的 users 表没有这些字段）
-- 注意：实际使用时需要检查 New API 的 users 表结构，如果已有则不需要添加
-- ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `phone` VARCHAR(20) COMMENT '手机号';
-- ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `real_name` VARCHAR(64) COMMENT '真实姓名';
-- ALTER TABLE `users` ADD COLUMN IF NOT EXISTS `id_card` VARCHAR(32) COMMENT '身份证号';

-- 渠道表扩展字段
-- ALTER TABLE `channels` ADD COLUMN IF NOT EXISTS `balance` DECIMAL(10,4) DEFAULT 0 COMMENT '渠道余额';
-- ALTER TABLE `channels` ADD COLUMN IF NOT EXISTS `daily_limit` DECIMAL(10,4) DEFAULT 0 COMMENT '日限额';
