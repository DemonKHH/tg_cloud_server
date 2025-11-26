package models

import (
	"database/sql/driver"
	"encoding/json"
)

// AgentScenario 智能体场景配置
type AgentScenario struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Topic       string        `json:"topic"`    // 全局话题/目标
	Duration    int           `json:"duration"` // 运行持续时间 (秒)
	Agents      []AgentConfig `json:"agents"`   // 参与的智能体
}

// AgentConfig 智能体配置
type AgentConfig struct {
	AccountID       uint64   `json:"account_id"`
	Persona         Persona  `json:"persona"`
	Goal            string   `json:"goal"`              // 个体目标
	ActiveRate      float64  `json:"active_rate"`       // 活跃度 (0.0-1.0)
	ImagePool       []string `json:"image_pool"`        // 图片资源池
	ImageGenEnabled bool     `json:"image_gen_enabled"` // 是否允许自动生成图片
}

// Persona 智能体人设
type Persona struct {
	Name       string   `json:"name"`
	Age        int      `json:"age"`
	Occupation string   `json:"occupation"`
	Style      []string `json:"style"`   // 说话风格
	Beliefs    []string `json:"beliefs"` // 核心观点
}

// Scan 实现 sql.Scanner 接口
func (as *AgentScenario) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, as)
}

// Value 实现 driver.Valuer 接口
func (as AgentScenario) Value() (driver.Value, error) {
	return json.Marshal(as)
}

// AgentDecisionRequest 智能体决策请求
type AgentDecisionRequest struct {
	ScenarioTopic   string                 `json:"scenario_topic"`
	AgentPersona    string                 `json:"agent_persona"`
	AgentGoal       string                 `json:"agent_goal"`
	ChatHistory     []ChatMessage          `json:"chat_history"`
	ImagePool       []string               `json:"image_pool"`
	ImageGenEnabled bool                   `json:"image_gen_enabled"`
	Context         map[string]interface{} `json:"context"`
}

// AgentDecisionResponse 智能体决策响应
type AgentDecisionResponse struct {
	ShouldSpeak  bool   `json:"should_speak"`
	Thought      string `json:"thought"`
	Action       string `json:"action"` // send_text, send_photo, generate_photo
	Content      string `json:"content"`
	MediaPath    string `json:"media_path,omitempty"`
	ImagePrompt  string `json:"image_prompt,omitempty"`
	ReplyToMsgID int64  `json:"reply_to_msg_id,omitempty"`
	DelaySeconds int    `json:"delay_seconds"`
}
