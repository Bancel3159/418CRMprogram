package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"crm-backend/pkg/response"
)

// HealthStatus 健康检查响应
type HealthStatus struct {
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	Database   string    `json:"database"`
	Redis      string    `json:"redis"`
	AppVersion string    `json:"version"`
}

const AppVersion = "0.1.0"

// healthCheck 健康检查
func (r *Router) healthCheck(c *gin.Context) {
	status := HealthStatus{
		Status:     "ok",
		Timestamp:  time.Now(),
		Database:   "unknown",
		Redis:      "unknown",
		AppVersion: AppVersion,
	}

	// 检查 PostgreSQL
	sqlDB, err := r.DB.DB()
	if err != nil {
		status.Database = "error: " + err.Error()
	} else if err := sqlDB.Ping(); err != nil {
		status.Database = "error: " + err.Error()
	} else {
		status.Database = "ok"
	}

	// 检查 Redis
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := r.RDB.Ping(ctx).Err(); err != nil {
		status.Redis = "error: " + err.Error()
	} else {
		status.Redis = "ok"
	}

	httpStatus := http.StatusOK
	if status.Database != "ok" || status.Redis != "ok" {
		status.Status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}

// 占位：避免 unused import 警告
var _ response.Response