package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeBroadcast MessageType = "broadcast" // 广播
	MessageTypeRemove    MessageType = "remove"    // 移除
	MessageTypeUpdate    MessageType = "update"    // 更新
	MessageTypeDirect    MessageType = "direct"    // 点对点
	MessageTypeAlert     MessageType = "alert"     // 告警(P8)
)

// Message WebSocket 消息
type Message struct {
	Type    MessageType `json:"type"`   // 消息类型
	Channel string      `json:"channel"` // 频道
	Event   string      `json:"event"`   // 事件名
	Data    interface{} `json:"data"`    // 数据
	Time    time.Time   `json:"time"`    // 时间戳
}

// Hub WebSocket Hub
type Hub struct {
	Clients    map[string]*Client // 所有客户端
	Broadcast  chan *BroadcastMsg // 广播消息队列
	Register   chan *Client       // 注册
	Unregister chan *Client       // 注销
	mu         sync.RWMutex
}

// BroadcastMsg 广播消息结构
type BroadcastMsg struct {
	Channel    string
	Message    []byte
	TargetRole string            // 可选：仅发给该角色（空 = 全部）
	ShouldSkip func(*Client) bool // 可选：每客户端判断是否跳过（true = 跳过）
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Broadcast:  make(chan *BroadcastMsg, 256),
		Register:   make(chan *Client, 64),
		Unregister: make(chan *Client, 64),
	}
}

// Run 运行 Hub（需在 goroutine 中）
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("[WS] client registered: id=%s user=%d role=%s", client.ID, client.UserID, client.Role)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] client unregistered: id=%s user=%d", client.ID, client.UserID)

		case msg := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.Clients {
				if !client.Channels[msg.Channel] {
					continue
				}
				if msg.TargetRole != "" && client.Role != msg.TargetRole {
					continue
				}
				if msg.ShouldSkip != nil && msg.ShouldSkip(client) {
					continue
				}
				select {
				case client.Send <- msg.Message:
				default:
					h.Unregister <- client
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToFilteredChannel 广播到指定频道 + 自定义过滤(每客户端判定)
// 典型用法:pool:broadcast 时跳过关闭接单开关的 follow 端用户
func (h *Hub) BroadcastToFilteredChannel(channel string, msgType MessageType, event string, data interface{}, shouldSkip func(*Client) bool) {
	msg := Message{
		Type:    msgType,
		Channel: channel,
		Event:   event,
		Data:    data,
		Time:    time.Now(),
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] marshal error: %v", err)
		return
	}
	h.Broadcast <- &BroadcastMsg{
		Channel:    channel,
		Message:    bytes,
		ShouldSkip: shouldSkip,
	}
}

// UpdateClientAcceptOrders 同步更新该用户所有在线客户端的接单状态(让 follow 端开关立即生效)
func (h *Hub) UpdateClientAcceptOrders(userID uint, acceptOrders bool) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.Clients {
		if c.UserID == userID {
			c.AcceptOrders = acceptOrders
			n++
		}
	}
	return n
}

// BroadcastToChannel 广播到指定频道
func (h *Hub) BroadcastToChannel(channel string, msgType MessageType, event string, data interface{}) {
	msg := Message{
		Type:    msgType,
		Channel: channel,
		Event:   event,
		Data:    data,
		Time:    time.Now(),
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] marshal error: %v", err)
		return
	}
	h.Broadcast <- &BroadcastMsg{
		Channel: channel,
		Message: bytes,
	}
}

// BroadcastToRole 广播到指定频道 + 指定角色
func (h *Hub) BroadcastToRole(channel, role string, msgType MessageType, event string, data interface{}) {
	msg := Message{
		Type:    msgType,
		Channel: channel,
		Event:   event,
		Data:    data,
		Time:    time.Now(),
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] marshal error: %v", err)
		return
	}
	h.Broadcast <- &BroadcastMsg{
		Channel:    channel,
		Message:    bytes,
		TargetRole: role,
	}
}

// BroadcastToFollowRoles 广播到 follow 端所有角色(b_manager / b_lead / b_staff)
// 不影响 admin(a_manager/a_lead/a_staff 不收,admin 走 web 端全局订单列表)
// 3 次入队(Hub 单消费者串行处理);若要严格保证时序可改为聚合一次 Send
func (h *Hub) BroadcastToFollowRoles(channel string, msgType MessageType, event string, data interface{}) {
	for _, role := range []string{"b_manager", "b_lead", "b_staff"} {
		h.BroadcastToRole(channel, role, msgType, event, data)
	}
}

// RegisterClient 注册客户端
func (h *Hub) RegisterClient(client *Client) {
	h.Register <- client
}

// UnregisterClient 注销客户端
func (h *Hub) UnregisterClient(client *Client) {
	h.Unregister <- client
}

// Stats 统计
func (h *Hub) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]interface{}{
		"total_clients": len(h.Clients),
	}
}

// ClientCount 客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}