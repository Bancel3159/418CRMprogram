package model

import (
	"time"

	"gorm.io/gorm"
)

// UploadTemplate 上传模板(按游戏 game_id 隔离,管理员可配置字段映射)
//
// column_mappings JSONB 结构:
//   {
//     "order_no":   "订单号",          // 系统字段 → Excel 列名
//     "player_id":  "玩家ID",
//     "amount":     "金额",
//     "currency":   "币种",
//     "status":     "支付状态",
//     "order_time": "下单时间",
//     "paid_at":    "支付时间",
//     "remark":     "备注"
//   }
// 未在映射中的列将整体写入 raw_payload 的 extras 字段
type UploadTemplate struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:100;not null" json:"name"` // 模板名(管理用)
	GameID uint  `gorm:"not null;index" json:"game_id"` // 所属游戏(per_game 策略)

	// 字段映射:列名 → 系统字段名
	ColumnMappings JSONB `gorm:"type:jsonb;not null;default:'{}'" json:"column_mappings"`

	// 日期格式与状态映射
	TimeFormat    string `gorm:"size:50;default:'2006-01-02 15:04:05'" json:"time_format"`      // 时间列解析格式(Go 参考时间)
	StatusMapping JSONB  `gorm:"type:jsonb;default:'{}'" json:"status_mapping"`                 // {"已支付":"paid","已退款":"refunded",...}
	Currency      string `gorm:"size:10;default:'CNY'" json:"currency"`                        // 默认币种

	// 元数据
	SampleFileURL string `gorm:"type:text" json:"sample_file_url"` // 示例文件 URL(/uploads/...)
	Remark        string `gorm:"type:text" json:"remark"`
	CreatedBy     *uint  `gorm:"index" json:"created_by"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Game    *Game `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Creator *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName 表名
func (UploadTemplate) TableName() string {
	return "upload_templates"
}