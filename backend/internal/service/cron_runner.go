package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/pkg/logger"
)

// CronRunner 合作方定时拉取调度器
type CronRunner struct {
	cron      *cron.Cron
	db        *gorm.DB
	mu        sync.Mutex
	jobByPartner map[uint]cron.EntryID // partnerID → entryID
	globalJobs    map[string]cron.EntryID // jobName → entryID(P11 全局任务)
}

// NewCronRunner 创建调度器
func NewCronRunner(db *gorm.DB) *CronRunner {
	c := cron.New(cron.WithSeconds(), cron.WithLogger(cron.DefaultLogger))
	return &CronRunner{
		cron:         c,
		db:           db,
		jobByPartner: make(map[uint]cron.EntryID),
		globalJobs:   make(map[string]cron.EntryID),
	}
}

// ReloadAll 重新加载所有 partner 的 cron 任务(启动时调用 + 增删 partner 后调用)
func (cr *CronRunner) ReloadAll(ctx context.Context) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// 清空旧任务
	for _, id := range cr.jobByPartner {
		cr.cron.Remove(id)
	}
	cr.jobByPartner = make(map[uint]cron.EntryID)

	// 加载启用 + 有 cron_expr 的 API 合作方
	var partners []model.Partner
	if err := cr.db.WithContext(ctx).
		Where("type = ? AND enabled = ? AND cron_expr <> ''", model.PartnerTypeAPI, true).
		Find(&partners).Error; err != nil {
		return fmt.Errorf("加载 partner 失败: %w", err)
	}

	for _, p := range partners {
		if err := cr.addJobLocked(&p); err != nil {
			logger.Warnf("partner %d cron 注册失败: %v", p.ID, err)
		} else {
			logger.Infof("partner %d (%s) cron 注册成功: %s", p.ID, p.Name, p.CronExpr)
		}
	}
	return nil
}

// ReloadOne 重新加载单个 partner 的任务
func (cr *CronRunner) ReloadOne(ctx context.Context, partnerID uint) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if oldID, ok := cr.jobByPartner[partnerID]; ok {
		cr.cron.Remove(oldID)
		delete(cr.jobByPartner, partnerID)
	}

	var p model.Partner
	if err := cr.db.WithContext(ctx).First(&p, partnerID).Error; err != nil {
		return err
	}
	if p.Type != model.PartnerTypeAPI || !p.Enabled || p.CronExpr == "" {
		return nil
	}
	return cr.addJobLocked(&p)
}

// addJobLocked 注册单个任务(需持有 mu)
func (cr *CronRunner) addJobLocked(p *model.Partner) error {
	partnerCopy := *p
	id, err := cr.cron.AddFunc(p.CronExpr, func() {
		cr.runOne(partnerCopy.ID)
	})
	if err != nil {
		return err
	}
	cr.jobByPartner[p.ID] = id
	return nil
}

// Start 启动调度器
func (cr *CronRunner) Start() {
	cr.cron.Start()
	logger.Info("cron 调度器启动")
}

// Stop 停止调度器
func (cr *CronRunner) Stop(ctx context.Context) {
	stopCtx := cr.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-ctx.Done():
	}
	logger.Info("cron 调度器停止")
}

// AddGlobalJob 注册全局 cron 任务(无 partner 关联,P11 跟进提醒用)
//   name     — 任务名(用于幂等去重)
//   expr     — cron 表达式(6-field with seconds,robfig/cron 格式)
//   jobFunc  — 执行函数(返回 error,异常不 panic)
//
// 同名任务重复注册会覆盖旧任务。expr 为空时跳过。
func (cr *CronRunner) AddGlobalJob(name, expr string, jobFunc func() error) error {
	if expr == "" {
		return nil
	}
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// 同名覆盖:先删旧的
	if oldID, ok := cr.globalJobs[name]; ok {
		cr.cron.Remove(oldID)
		delete(cr.globalJobs, name)
	}

	id, err := cr.cron.AddFunc(expr, func() {
		start := time.Now()
		if err := jobFunc(); err != nil {
			logger.Errorf("global job [%s] 失败(cost=%v): %v", name, time.Since(start), err)
		} else {
			logger.Infof("global job [%s] 成功(cost=%v)", name, time.Since(start))
		}
	})
	if err != nil {
		return err
	}
	cr.globalJobs[name] = id
	logger.Infof("global job [%s] 已注册: %s", name, expr)
	return nil
}

// RunGlobalJobNow 立即执行指定全局任务(供 API 触发 — 调试 / 立刻生效用)
func (cr *CronRunner) RunGlobalJobNow(name string) bool {
	cr.mu.Lock()
	id, ok := cr.globalJobs[name]
	cr.mu.Unlock()
	if !ok {
		return false
	}
	entries := cr.cron.Entries()
	for _, e := range entries {
		if e.ID == id {
			go e.Job.Run()
			return true
		}
	}
	return false
}

// runOne 执行一次(供 cron 调用,异常不 panic)
// 此处简化:实际拉取逻辑在 api 包,通过回调注入
// 为避免循环依赖,这里使用接口注入
type PullFunc func(ctx context.Context, partnerID uint) error

// 注入拉取函数
var pullFunc PullFunc

// SetPullFunc 设置拉取回调(api 包启动时调用)
func SetPullFunc(f PullFunc) {
	pullFunc = f
}

func (cr *CronRunner) runOne(partnerID uint) {
	if pullFunc == nil {
		logger.Warnf("pullFunc 未注入,partner %d 跳过", partnerID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	start := time.Now()
	err := pullFunc(ctx, partnerID)
	cost := time.Since(start)
	if err != nil {
		logger.Errorf("partner %d cron 拉取失败(cost=%v): %v", partnerID, cost, err)
	} else {
		logger.Infof("partner %d cron 拉取成功(cost=%v)", partnerID, cost)
	}
}