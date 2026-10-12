-- 初始化数据库
-- 启用必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 设置时区
SET timezone = 'Asia/Shanghai';

-- 注：业务表后续通过 GORM AutoMigrate 自动迁移创建
-- 这个文件只用于初始化扩展和默认设置