package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crm-backend/internal/model"
	"crm-backend/internal/ws"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"     // 普通通知
	AlertLevelWarning  AlertLevel = "warning"  // 警告
	AlertLevelCritical AlertLevel = "critical" // 严重
)

// Alert 告警结构(WebSocket 消息 payload)
type Alert struct {
	ID        string      `json:"id"`
	Level     AlertLevel  `json:"level"`
	Title     string      `json:"title"`
	Message   string      `json:"message"`
	Link      string      `json:"link,omitempty"`
	Extension interface{} `json:"extension,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

// AlertNotifier 告警推送器
type AlertNotifier struct {
	DB  *gorm.DB
	Hub *ws.Hub
}

// NewAlertNotifier 创建告警推送器
func NewAlertNotifier(db *gorm.DB, hub *ws.Hub) *AlertNotifier {
	return &AlertNotifier{DB: db, Hub: hub}
}

// Send 推送一条告警到 admin 频道
func (n *AlertNotifier) Send(alert Alert) {
	if n.Hub == nil {
		return
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}
	if alert.ID == "" {
		alert.ID = time.Now().Format("20060102150405.000000")
	}
	n.Hub.BroadcastToChannel("alert:admin", ws.MessageTypeAlert, "alert", alert)
}

// SendFromContext 从 gin context 构造(可获取用户信息)
func (n *AlertNotifier) SendFromContext(c *gin.Context, alert Alert) {
	n.Send(alert)
}

// PushOrderImported 推送新订单入库告警
func (n *AlertNotifier) PushOrderImported(orderNo string, amount float64, source model.OrderSource, attributionStatus model.AttributionStatus) {
	level := AlertLevelInfo
	if attributionStatus == model.AttributionPending {
		level = AlertLevelWarning
	}
	n.Send(Alert{
		Level:   level,
		Title:   "新订单入库",
		Message: orderNo + " / ¥" + FormatAmountFloat64(amount) + " / " + string(source),
		Link:    "/orders?attribution_status=" + string(attributionStatus),
	})
}

// PushLoginFailed 推送登录失败告警(连续失败 3 次)
func (n *AlertNotifier) PushLoginFailed(username, ip string, count int) {
	level := AlertLevelWarning
	if count >= 5 {
		level = AlertLevelCritical
	}
	n.Send(Alert{
		Level:   level,
		Title:   "登录失败告警",
		Message: username + " 从 " + ip + " 失败 " + IntToString(count) + " 次",
		Link:    "/system/audit?status=failure&username=" + username,
	})
}

// FormatAmountFloat64 格式化金额为 "1234.50"
func FormatAmountFloat64(v float64) string {
	whole := int64(v)
	frac := int64((v - float64(whole)) * 100)
	if frac < 0 {
		frac = -frac
	}
	if whole < 0 {
		whole = -whole
	}
	return UintToString(uint(whole)) + "." + padLeft(UintToString(uint(frac)), 2, '0')
}

// UintToString uint 转字符串(避免引入 strconv 包别名)
func UintToString(n uint) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// IntToString int 转字符串
func IntToString(n int) string {
	negative := n < 0
	abs := uint(0)
	if negative {
		abs = uint(-n)
	} else {
		abs = uint(n)
	}
	s := UintToString(abs)
	if negative {
		s = "-" + s
	}
	return s
}

func padLeft(s string, n int, c byte) string {
	for len(s) < n {
		s = string(c) + s
	}
	return s
}