-- 创建用户表（匹配GORM结构体定义）
CREATE TABLE IF NOT EXISTS users (
                                     `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户主键ID',
                                     `uuid` VARCHAR(36) NOT NULL COMMENT '用户UUID（全局唯一标识）',
    `username` VARCHAR(50) NOT NULL COMMENT '用户名（登录用）',
    `email` VARCHAR(100) DEFAULT NULL COMMENT '用户邮箱',
    `phone` VARCHAR(20) DEFAULT NULL COMMENT '用户手机号',
    `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希（bcrypt加密后）',
    `role` VARCHAR(20) NOT NULL DEFAULT 'candidate' COMMENT '用户角色：candidate（候选人）、hr（HR）、admin（管理员）',
    `avatar_url` VARCHAR(500) DEFAULT NULL COMMENT '用户头像URL',
    `real_name` VARCHAR(50) DEFAULT NULL COMMENT '用户真实姓名',
    `gender` VARCHAR(10) DEFAULT NULL COMMENT '性别：male（男）、female（女）、other（其他）',
    `birth_date` DATE DEFAULT NULL COMMENT '出生日期（格式：YYYY-MM-DD）',
    -- 主键与唯一索引（匹配GORM标签）
    PRIMARY KEY (`id`),
    UNIQUE INDEX `idx_uuid` (`uuid`),
    UNIQUE INDEX `idx_username` (`username`),
    UNIQUE INDEX `idx_email` (`email`),
    UNIQUE INDEX `idx_phone` (`phone`)
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '用户信息表';