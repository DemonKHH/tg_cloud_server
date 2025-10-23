package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ProxyProtocol 代理协议枚举
type ProxyProtocol string

const (
	ProxyHTTP   ProxyProtocol = "http"
	ProxyHTTPS  ProxyProtocol = "https"
	ProxySOCKS5 ProxyProtocol = "socks5"
)

// ProxyStatus 代理状态枚举
type ProxyStatus string

const (
	StatusActive   ProxyStatus = "active"
	StatusInactive ProxyStatus = "inactive"
	StatusError    ProxyStatus = "error"
	StatusTesting  ProxyStatus = "testing"
	StatusUntested ProxyStatus = "untested"
)

// ProxyIP 代理IP模型（客户自管理）
type ProxyIP struct {
	ID          uint64        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64        `json:"user_id" gorm:"not null;index"`   // 归属用户
	Name        string        `json:"name" gorm:"size:100"`            // 代理名称/备注
	Host        string        `json:"host" gorm:"size:255;not null"`   // 主机地址 (IP或域名)
	IP          string        `json:"ip" gorm:"size:45;not null"`      // IP地址
	Port        int           `json:"port" gorm:"not null"`            // 端口
	Protocol    ProxyProtocol `json:"protocol" gorm:"type:enum('http','https','socks5');not null"`
	Username    string        `json:"username" gorm:"size:100"`                                   // 代理用户名
	Password    string        `json:"-" gorm:"size:100"`                                          // 代理密码（隐藏）
	Country     string        `json:"country" gorm:"size:10"`                                     // 国家代码
	Status      ProxyStatus   `json:"status" gorm:"type:enum('active','inactive','error','testing','untested');default:'untested'"` // 代理状态
	IsActive    bool          `json:"is_active" gorm:"default:true"`                              // 是否启用
	SuccessRate float64       `json:"success_rate" gorm:"type:decimal(5,2);default:0.00"`         // 成功率
	AvgLatency  int           `json:"avg_latency"`                                                // 平均延迟(ms)
	LastTestAt  *time.Time    `json:"last_test_at"`                                               // 最后测试时间
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	// 关联关系
	User     User        `json:"user" gorm:"foreignKey:UserID"`
	Accounts []TGAccount `json:"accounts" gorm:"foreignKey:ProxyID"`
}

// TableName 指定表名
func (ProxyIP) TableName() string {
	return "proxy_ips"
}

// GetAddress 获取代理地址
func (p *ProxyIP) GetAddress() string {
	return fmt.Sprintf("%s://%s:%d", p.Protocol, p.IP, p.Port)
}

// GetAuthAddress 获取带认证的代理地址
func (p *ProxyIP) GetAuthAddress() string {
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", p.Protocol, p.Username, p.Password, p.IP, p.Port)
	}
	return p.GetAddress()
}

// IsHealthy 检查代理是否健康
func (p *ProxyIP) IsHealthy() bool {
	return p.IsActive && p.SuccessRate >= 80.0 && p.AvgLatency < 5000
}

// GetQualityLevel 获取代理质量等级
func (p *ProxyIP) GetQualityLevel() string {
	if !p.IsActive {
		return "disabled"
	}

	if p.SuccessRate >= 95.0 && p.AvgLatency < 1000 {
		return "excellent"
	} else if p.SuccessRate >= 90.0 && p.AvgLatency < 2000 {
		return "good"
	} else if p.SuccessRate >= 80.0 && p.AvgLatency < 5000 {
		return "average"
	} else {
		return "poor"
	}
}

// UpdateStats 更新统计信息
func (p *ProxyIP) UpdateStats(success bool, latency int) {
	// 这里应该实现统计更新逻辑
	// 可以使用滑动窗口算法来计算成功率和平均延迟
	p.LastTestAt = &time.Time{}
	*p.LastTestAt = time.Now()
}

// BeforeCreate 创建前钩子
func (p *ProxyIP) BeforeCreate(tx *gorm.DB) error {
	p.IsActive = true
	p.SuccessRate = 0.0
	return nil
}

// ProxyConfig 代理配置（用于Telegram客户端）
type ProxyConfig struct {
	Protocol ProxyProtocol `json:"protocol"`
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Username string        `json:"username,omitempty"`
	Password string        `json:"password,omitempty"`
}

// CreateProxyRequest 创建代理请求
type CreateProxyRequest struct {
	Name     string        `json:"name" binding:"required"`
	Host     string        `json:"host" binding:"required"`
	IP       string        `json:"ip" binding:"required,ip"`
	Port     int           `json:"port" binding:"required,min=1,max=65535"`
	Protocol ProxyProtocol `json:"protocol" binding:"required,oneof=http https socks5"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	Country  string        `json:"country"`
}

// UpdateProxyRequest 更新代理请求
type UpdateProxyRequest struct {
	Name     string        `json:"name"`
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Protocol ProxyProtocol `json:"protocol"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	Country  string        `json:"country"`
	IsActive *bool         `json:"is_active"`
}

// ProxyTestResult 代理测试结果
type ProxyTestResult struct {
	ProxyID    uint64    `json:"proxy_id"`
	Success    bool      `json:"success"`
	Latency    int       `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	TestedAt   time.Time `json:"tested_at"`
	IPLocation string    `json:"ip_location,omitempty"`
}

// ProxyDetail 代理详细统计信息
type ProxyDetail struct {
	ProxyID      uint64     `json:"proxy_id"`
	Name         string     `json:"name"`
	Address      string     `json:"address"`
	SuccessRate  float64    `json:"success_rate"`
	AvgLatency   int        `json:"avg_latency_ms"`
	QualityLevel string     `json:"quality_level"`
	AccountCount int64      `json:"account_count"`
	LastTestAt   *time.Time `json:"last_test_at"`
	IsHealthy    bool       `json:"is_healthy"`
}

// BatchProxyTestRequest 批量代理测试请求
type BatchProxyTestRequest struct {
	ProxyIDs []uint64 `json:"proxy_ids" binding:"required"`
}

// BindProxyRequest 绑定代理请求
type BindProxyRequest struct {
	AccountID uint64  `json:"account_id" binding:"required"`
	ProxyID   *uint64 `json:"proxy_id"` // nil表示取消绑定
}
