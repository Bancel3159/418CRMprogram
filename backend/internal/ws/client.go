package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client WebSocket 客户端
type Client struct {
	ID           string          // 客户端唯一 ID
	UserID       uint            // 用户 ID
	Username     string          // 用户名
	Role         string          // 角色
	DeptID       *uint           // 部门 ID
	Channels     map[string]bool // 订阅的频道
	Conn         *websocket.Conn // 连接
	Send         chan []byte     // 发送队列
	Hub          *Hub            // 所属 hub
	AcceptOrders bool            // 是否接单（follow 端公开池广播接收开关；默认 true 即不阻拦）
	mu           sync.Mutex      // 保护 closed
	closed       bool            // 是否已关闭
}

// NewClient 创建客户端
func NewClient(id string, userID uint, username, role string, deptID *uint, channels map[string]bool, conn *websocket.Conn, hub *Hub, acceptOrders bool) *Client {
	return &Client{
		ID:           id,
		UserID:       userID,
		Username:     username,
		Role:         role,
		DeptID:       deptID,
		Channels:     channels,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Hub:          hub,
		AcceptOrders: acceptOrders,
	}
}

// Start 启动读写 goroutine
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// readPump 读取客户端消息
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] read error for client %s: %v", c.ID, err)
			}
			break
		}
	}
}

// writePump 向客户端发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendJSON 发送 JSON 消息
func (c *Client) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.Send <- bytes:
		return nil
	default:
		return nil // 队列满，丢弃
	}
}