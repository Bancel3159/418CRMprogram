package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"crm-backend/internal/model"
	"crm-backend/pkg/logger"
)

// ============ P11 跟进规则引擎 ============

// FollowConfig 运行时读取的阈值(从 system_configs + 默认值)
type FollowConfig struct {
	NewRegisterDays    int    // 新注册跟进 X 天(默认 3)
	PaidOfflineDays    int    // 付费玩家离线多少天升级提醒(默认 2)
	PaidOfflineMaxDays int    // 付费玩家离线多少天自动流失(默认 5)
	DailyCron          string // 每日扫描 cron 表达式(默认 "0 0 8 * * *" UTC 8:00)
}

// DefaultFollowConfig 默认值(无 system_configs 行时使用)
func DefaultFollowConfig() FollowConfig {
	return FollowConfig{
		NewRegisterDays:    3,
		PaidOfflineDays:    2,
		PaidOfflineMaxDays: 5,
		DailyCron:          "0 0 8 * * *", // UTC 8:00
	}
}

// LoadFollowConfig 从 system_configs 读配置(缺失项用默认值)
func LoadFollowConfig(db *gorm.DB) FollowConfig {
	cfg := DefaultFollowConfig()
	rows := []model.SystemConfig{}
	if err := db.Where("key IN ?", []string{
		model.FollowConfigNewRegisterDays,
		model.FollowConfigPaidOfflineDays,
		model.FollowConfigPaidOfflineMaxDays,
		model.FollowConfigDailyCron,
	}).Find(&rows).Error; err != nil {
		logger.Warnf("LoadFollowConfig 失败,用默认值: %v", err)
		return cfg
	}
	for _, r := range rows {
		switch r.Key {
		case model.FollowConfigNewRegisterDays:
			if v, err := strconv.Atoi(r.Value); err == nil && v > 0 && v <= 30 {
				cfg.NewRegisterDays = v
			}
		case model.FollowConfigPaidOfflineDays:
			if v, err := strconv.Atoi(r.Value); err == nil && v > 0 && v <= 30 {
				cfg.PaidOfflineDays = v
			}
		case model.FollowConfigPaidOfflineMaxDays:
			if v, err := strconv.Atoi(r.Value); err == nil && v > 0 && v <= 90 {
				cfg.PaidOfflineMaxDays = v
			}
		case model.FollowConfigDailyCron:
			if r.Value != "" {
				cfg.DailyCron = r.Value
			}
		}
	}
	return cfg
}

// EnsureDefaultConfigs 把默认配置写入 system_configs(启动时调用一次)
func EnsureDefaultConfigs(db *gorm.DB) {
	defaults := []model.SystemConfig{
		{Key: model.FollowConfigNewRegisterDays, Value: "3", Description: "新注册跟进天数(超过自动流失)"},
		{Key: model.FollowConfigPaidOfflineDays, Value: "2", Description: "付费玩家离线多少天升级提醒优先级"},
		{Key: model.FollowConfigPaidOfflineMaxDays, Value: "5", Description: "付费玩家离线多少天自动流失"},
		{Key: model.FollowConfigDailyCron, Value: "0 0 8 * * *", Description: "每日扫描 cron(UTC,6-field with seconds)"},
	}
	for _, c := range defaults {
		// ON CONFLICT DO NOTHING(避免重复)
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&c).Error; err != nil {
			logger.Warnf("EnsureDefaultConfigs %s 失败: %v", c.Key, err)
		}
	}
}

// FollowReminderEngine 跟进提醒引擎
type FollowReminderEngine struct {
	db  *gorm.DB
	cfg FollowConfig
}

// NewFollowReminderEngine 构造 + 自动加载配置
func NewFollowReminderEngine(db *gorm.DB) *FollowReminderEngine {
	return &FollowReminderEngine{
		db:  db,
		cfg: LoadFollowConfig(db),
	}
}

// Reload 重新加载配置(配置变更后调用)
func (e *FollowReminderEngine) Reload() {
	e.cfg = LoadFollowConfig(e.db)
}

// ReloadCronExpr 返回当前 cron 表达式(供 main.go 注册到 CronRunner)
func (e *FollowReminderEngine) ReloadCronExpr() string {
	e.Reload()
	return e.cfg.DailyCron
}

// DailyScan 每日扫描 — 由 cron 触发
// 步骤:
//   1. 清掉今天之前的 pending 提醒(expired)
//   2. 对每个非流失 lead 计算今日 reminders(upsert)
//   3. 自动流失判定(超过 X 天未跟进的新注册 / 离线超 5 天的付费)
//   4. 返回统计
func (e *FollowReminderEngine) DailyScan(ctx context.Context) (DailyScanResult, error) {
	e.Reload()
	now := time.Now()
	today := startOfDay(now)

	result := DailyScanResult{}

	// ============ 1. 清掉过期的 pending 提醒 ============
	res := e.db.WithContext(ctx).
		Model(&model.LeadReminder{}).
		Where("status = ? AND remind_date < ?", model.ReminderStatusPending, today).
		Update("status", model.ReminderStatusExpired)
	if res.Error != nil {
		return result, res.Error
	}
	result.ExpiredReminders = res.RowsAffected

	// ============ 2. 找出需要扫描的 lead ============
	// 只看非 archived(archived = 已流失归档,不再生成提醒)
	var leads []model.Lead
	if err := e.db.WithContext(ctx).
		Where("status <> ?", model.LeadStatusArchived).
		Find(&leads).Error; err != nil {
		return result, err
	}

	// ============ 3. 计算 is_paid(paid_mark OR paid_auto) ============
	paidMap, err := e.computePaidSet(ctx)
	if err != nil {
		return result, err
	}

	// ============ 4. 对每个 lead 生成今日提醒(upsert) ============
	for i := range leads {
		lead := &leads[i]
		lead.IsPaid = paidMap[lead.PlayerID]

		// 4a. 新注册跟进 X 天
		if !lead.IsLost && !lead.IsPaid {
			daysSinceCreate := daysBetween(lead.CreatedAt, now)
			if daysSinceCreate <= e.cfg.NewRegisterDays {
				if err := e.upsertReminder(ctx, lead.ID, today, model.ReminderFollowNewRegister,
					model.ReminderPriorityNormal, ""); err == nil {
					result.NewReminders++
				}
			}
		}

		// 4b. 已付费每日跟进
		if !lead.IsLost && lead.IsPaid {
			if err := e.upsertReminder(ctx, lead.ID, today, model.ReminderFollowPaidDaily,
				model.ReminderPriorityNormal, ""); err == nil {
				result.PaidDailyReminders++
			}

			// 4c. 付费玩家离线升级
			lastActive := e.getLastActive(lead)
			offlineDays := daysBetween(lastActive, now)
			if offlineDays >= e.cfg.PaidOfflineDays {
				priority := offlineDays - e.cfg.PaidOfflineDays // 2 天 = 0,3 天 = 1,4 天+ = 2
				if priority > int(model.ReminderPriorityHigh) {
					priority = int(model.ReminderPriorityHigh)
				}
				note := "已离线 " + strconv.Itoa(offlineDays) + " 天"
				if err := e.upsertReminder(ctx, lead.ID, today, model.ReminderFollowPaidOffline,
					model.ReminderPriority(priority), note); err == nil {
					result.OfflineAlerts++
				}
			}
		}
	}

	// ============ 5. 自动流失判定 ============
	// 5a. 新注册跟进超过 X 天 + 从未付费 + 今日 reminders 全部未跟进 → 流失
	expireDate := today.AddDate(0, 0, -e.cfg.NewRegisterDays)
	res2 := e.db.WithContext(ctx).
		Model(&model.Lead{}).
		Where("is_lost = ? AND status <> ? AND paid_mark = ?", false, model.LeadStatusArchived, false).
		Where("created_at <= ?", expireDate).
		UpdateColumns(map[string]interface{}{
			"is_lost":     true,
			"lost_at":     today,
			"lost_reason": "auto_new_register_timeout",
		})
	if res2.Error != nil {
		return result, res2.Error
	}
	result.AutoLostNewRegister = res2.RowsAffected

	// 5b. 付费玩家离线超过 MaxDays → 流失
	lostDate := today.AddDate(0, 0, -e.cfg.PaidOfflineMaxDays)
	// 用 subquery 限定:is_paid 的 lead 且 last_player_active_at < 流失日期
	if len(paidMap) > 0 {
		// 简化:用 paid_mark=true 的 lead(因为 paid_auto 需要 join order 表)
		res3 := e.db.WithContext(ctx).
			Model(&model.Lead{}).
			Where("is_lost = ? AND status <> ? AND paid_mark = ?", false, model.LeadStatusArchived, true).
			Where("last_player_active_at IS NOT NULL AND last_player_active_at < ?", lostDate).
			UpdateColumns(map[string]interface{}{
				"is_lost":     true,
				"lost_at":     today,
				"lost_reason": "auto_paid_offline",
			})
		if res3.Error != nil {
			return result, res3.Error
		}
		result.AutoLostPaidOffline = res3.RowsAffected
	}

	logger.Infof("FollowReminderEngine.DailyScan 完成: 今日新建=%d 付费每日=%d 离线升级=%d 过期清理=%d 自动流失(新注册)=%d 自动流失(付费)=%d",
		result.NewReminders, result.PaidDailyReminders, result.OfflineAlerts,
		result.ExpiredReminders, result.AutoLostNewRegister, result.AutoLostPaidOffline)
	return result, nil
}

// DailyScanResult 扫描结果统计
type DailyScanResult struct {
	NewReminders         int64 `json:"new_reminders"`           // 今日新建提醒(新注册跟进)
	PaidDailyReminders   int64 `json:"paid_daily_reminders"`    // 付费玩家每日跟进
	OfflineAlerts        int64 `json:"offline_alerts"`          // 离线升级提醒
	ExpiredReminders     int64 `json:"expired_reminders"`       // 过期清理
	AutoLostNewRegister  int64 `json:"auto_lost_new_register"`  // 自动流失(新注册超时)
	AutoLostPaidOffline  int64 `json:"auto_lost_paid_offline"`  // 自动流失(付费离线)
}

// computePaidSet 计算 player_id -> 是否付费 的集合
// paid = paid_mark=true OR orders 表里 status='paid'
func (e *FollowReminderEngine) computePaidSet(ctx context.Context) (map[string]bool, error) {
	out := make(map[string]bool)

	// 来源 1:paid_mark=true 的 leads
	type paidMarkRow struct {
		PlayerID string
	}
	var marks []paidMarkRow
	if err := e.db.WithContext(ctx).
		Model(&model.Lead{}).
		Select("DISTINCT player_id").
		Where("paid_mark = ?", true).
		Scan(&marks).Error; err != nil {
		return nil, err
	}
	for _, r := range marks {
		out[r.PlayerID] = true
	}

	// 来源 2:orders 表 status=paid
	type orderRow struct {
		PlayerID string
	}
	var orders []orderRow
	if err := e.db.WithContext(ctx).
		Model(&model.Order{}).
		Select("DISTINCT player_id").
		Where("status = ?", model.OrderStatusPaid).
		Scan(&orders).Error; err != nil {
		return nil, err
	}
	for _, r := range orders {
		out[r.PlayerID] = true
	}

	return out, nil
}

// upsertReminder 写入提醒(唯一约束:lead_id+remind_date+remind_type — 重复则更新 priority/note/status 重置)
func (e *FollowReminderEngine) upsertReminder(ctx context.Context, leadID uint, date time.Time,
	typ model.ReminderType, priority model.ReminderPriority, note string) error {

	date = startOfDay(date)

	// 先查现有
	var existing model.LeadReminder
	err := e.db.WithContext(ctx).
		Where("lead_id = ? AND remind_date = ? AND remind_type = ?", leadID, date, typ).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 新建
		r := &model.LeadReminder{
			LeadID:     leadID,
			RemindDate: date,
			RemindType: typ,
			Priority:   priority,
			Status:     model.ReminderStatusPending,
			Note:       note,
		}
		return e.db.WithContext(ctx).Create(r).Error
	}
	if err != nil {
		return err
	}
	// 已存在 + 状态是 done/expired/dismissed → 不再复活(避免覆盖已完成记录)
	if existing.Status != model.ReminderStatusPending {
		return nil
	}
	// 已存在 + pending → 更新 priority 和 note(可能离线天数增加)
	updates := map[string]interface{}{
		"priority": priority,
		"note":     note,
	}
	return e.db.WithContext(ctx).Model(&existing).Updates(updates).Error
}

// getLastActive 取玩家最后活跃时间(优先 last_player_active_at,fallback 到 last_follow_at,再 fallback created_at)
func (e *FollowReminderEngine) getLastActive(lead *model.Lead) time.Time {
	if lead.LastPlayerActiveAt != nil {
		return *lead.LastPlayerActiveAt
	}
	if lead.LastFollowAt != nil {
		return *lead.LastFollowAt
	}
	return lead.CreatedAt
}

// startOfDay 取日期 00:00:00
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// daysBetween 计算两个时间相差的整天数(t1 早,t2 晚)
func daysBetween(t1, t2 time.Time) int {
	d1 := startOfDay(t1)
	d2 := startOfDay(t2)
	return int(d2.Sub(d1).Hours() / 24)
}

// ============ 手动 API 调用方法 ============

// CompleteReminder 标记提醒完成(用户写了跟进记录后调)
func (e *FollowReminderEngine) CompleteReminder(ctx context.Context, leadID uint, completedBy uint, date time.Time) error {
	today := startOfDay(date)
	return e.db.WithContext(ctx).
		Model(&model.LeadReminder{}).
		Where("lead_id = ? AND remind_date = ? AND status = ?", leadID, today, model.ReminderStatusPending).
		Updates(map[string]interface{}{
			"status":       model.ReminderStatusDone,
			"completed_at": time.Now(),
			"completed_by": completedBy,
		}).Error
}

// ManualMarkLost 手动标记流失(管理员 / 主管触发)
func (e *FollowReminderEngine) ManualMarkLost(ctx context.Context, leadID uint) error {
	return e.db.WithContext(ctx).
		Model(&model.Lead{}).
		Where("id = ?", leadID).
		UpdateColumns(map[string]interface{}{
			"is_lost":     true,
			"lost_at":     time.Now(),
			"lost_reason": "manual",
		}).Error
}

// ManualMarkActive 手动标记激活(用于"再次登录会激活"的手动兜底;Phase 3 自动激活)
func (e *FollowReminderEngine) ManualMarkActive(ctx context.Context, leadID uint) error {
	return e.db.WithContext(ctx).
		Model(&model.Lead{}).
		Where("id = ?", leadID).
		UpdateColumns(map[string]interface{}{
			"is_lost":        false,
			"reactivated_at": time.Now(),
			"lost_at":        nil,
			"lost_reason":    "",
		}).Error
}
