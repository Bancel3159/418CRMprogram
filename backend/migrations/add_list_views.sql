-- P10 列表视图配置（玩家 / 订单 列表字段可配置）
-- 该表由 GORM AutoMigrate 自动创建，本文件作为 schema 参考文档。
-- 实际建表通过 cmd/migrate/main.go 触发。

CREATE TABLE IF NOT EXISTS list_views (
    id          BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(30) NOT NULL,                  -- 'player' | 'order'
    name        VARCHAR(50) NOT NULL,                  -- 视图名（同 entity 内唯一，name 重复检查放在应用层）
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,        -- 列表页默认视图
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,        -- 系统视图（不可删）
    fields      JSONB NOT NULL DEFAULT '[]'::jsonb,    -- 字段配置数组 [{code,label,source,width,align,formatter}, ...]
    created_by  BIGINT REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_list_views_entity        ON list_views(entity_type);
CREATE INDEX IF NOT EXISTS idx_list_views_entity_default ON list_views(entity_type, is_default) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_list_views_created_by   ON list_views(created_by);
CREATE INDEX IF NOT EXISTS idx_list_views_deleted_at   ON list_views(deleted_at);

-- 种子：每个 entity 一个默认视图（由 cmd/seed 写入）
-- player 默认视图 → 11 个字段（game_code, player_id, latest_username, latest_country, latest_is_target, lead_count, order_count, paid_order_count, total_order_amount, last_follow_at, last_lead_at）
-- order 默认视图  → 11 个字段（order_no, player_id, game_code, amount, status, source, attribution_status, is_first_charge, lead_player_country, lead_creator, order_time）