package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"grotto-monitor/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type OfflineMessage struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	Time      time.Time `json:"time"`
	Delivered bool      `json:"delivered"`
}

type ClientInfo struct {
	conn     *websocket.Conn
	connectedAt time.Time
	lastAck     int64
}

type Manager struct {
	clients         map[*websocket.Conn]*ClientInfo
	mu              sync.RWMutex
	broadcast       chan models.WebSocketMessage
	offlineQueue    []OfflineMessage
	offlineMu       sync.RWMutex
	maxOfflineMsgs  int
	msgCounter      int64
	counterMu       sync.Mutex
}

var (
	wsManager *Manager
	once      sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		wsManager = &Manager{
			clients:        make(map[*websocket.Conn]*ClientInfo),
			broadcast:      make(chan models.WebSocketMessage, 256),
			offlineQueue:   make([]OfflineMessage, 0, 500),
			maxOfflineMsgs: 500,
		}
		go wsManager.run()
		go wsManager.cleanupOfflineMessages()
	})
	return wsManager
}

func (m *Manager) nextMsgID() int64 {
	m.counterMu.Lock()
	defer m.counterMu.Unlock()
	m.msgCounter++
	return m.msgCounter
}

func (m *Manager) run() {
	for msg := range m.broadcast {
		offlineMsg := OfflineMessage{
			ID:      m.nextMsgID(),
			Type:    msg.Type,
			Payload: msg.Payload,
			Time:    msg.Time,
		}

		m.mu.RLock()
		clientCount := len(m.clients)
		m.mu.RUnlock()

		if clientCount > 0 {
			delivered := false
			m.mu.RLock()
			for _, info := range m.clients {
				err := info.conn.WriteJSON(msg)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					go func(c *websocket.Conn) {
						m.mu.Lock()
						delete(m.clients, c)
						c.Close()
						m.mu.Unlock()
					}(info.conn)
					continue
				}
				delivered = true
			}
			m.mu.RUnlock()
			offlineMsg.Delivered = delivered
		}

		if !offlineMsg.Delivered {
			m.enqueueOffline(offlineMsg)
		} else if msg.Type == "ALERT" {
			m.enqueueOffline(offlineMsg)
		}
	}
}

func (m *Manager) enqueueOffline(msg OfflineMessage) {
	m.offlineMu.Lock()
	defer m.offlineMu.Unlock()

	m.offlineQueue = append(m.offlineQueue, msg)
	if len(m.offlineQueue) > m.maxOfflineMsgs {
		m.offlineQueue = m.offlineQueue[len(m.offlineQueue)-m.maxOfflineMsgs:]
	}
}

func (m *Manager) cleanupOfflineMessages() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.offlineMu.Lock()
		cutoff := time.Now().Add(-24 * time.Hour)
		filtered := make([]OfflineMessage, 0, len(m.offlineQueue))
		for _, msg := range m.offlineQueue {
			if msg.Time.After(cutoff) {
				filtered = append(filtered, msg)
			}
		}
		m.offlineQueue = filtered
		m.offlineMu.Unlock()
	}
}

func (m *Manager) deliverOfflineMessages(conn *websocket.Conn) int {
	m.offlineMu.RLock()
	defer m.offlineMu.RUnlock()

	delivered := 0
	for _, msg := range m.offlineQueue {
		wsMsg := models.WebSocketMessage{
			Type:    "OFFLINE_" + msg.Type,
			Payload: msg.Payload,
			Time:    msg.Time,
		}
		err := conn.WriteJSON(wsMsg)
		if err != nil {
			log.Printf("Failed to deliver offline message %d: %v", msg.ID, err)
			break
		}
		delivered++
	}
	return delivered
}

func (m *Manager) HandleConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	m.mu.Lock()
	m.clients[conn] = &ClientInfo{
		conn:        conn,
		connectedAt: time.Now(),
	}
	connCount := len(m.clients)
	m.mu.Unlock()

	log.Printf("WebSocket client connected, total: %d", connCount)

	offlinedCount := m.deliverOfflineMessages(conn)
	if offlinedCount > 0 {
		log.Printf("Delivered %d offline messages to new client", offlinedCount)
	}

	defer func() {
		m.mu.Lock()
		delete(m.clients, conn)
		remaining := len(m.clients)
		m.mu.Unlock()
		conn.Close()
		log.Printf("WebSocket client disconnected, remaining: %d", remaining)
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			m.mu.RLock()
			info, exists := m.clients[conn]
			m.mu.RUnlock()
			if !exists {
				return
			}
			if err := info.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var clientMsg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &clientMsg); err == nil {
			if msgType, ok := clientMsg["type"].(string); ok && msgType == "ACK" {
				if msgID, ok := clientMsg["msgId"].(float64); ok {
					m.acknowledgeMessage(int64(msgID))
				}
			}
		}
	}
}

func (m *Manager) acknowledgeMessage(msgID int64) {
	m.offlineMu.Lock()
	defer m.offlineMu.Unlock()
	for i := range m.offlineQueue {
		if m.offlineQueue[i].ID == msgID {
			m.offlineQueue[i].Delivered = true
			break
		}
	}
}

func (m *Manager) Broadcast(msgType string, payload interface{}) {
	msg := models.WebSocketMessage{
		Type:    msgType,
		Payload: payload,
		Time:    time.Now(),
	}
	select {
	case m.broadcast <- msg:
	default:
		log.Println("WebSocket broadcast channel full, dropping message")
	}
}

func (m *Manager) BroadcastAlert(alert *models.Alert) {
	alertJSON, _ := json.Marshal(alert)
	var alertMap map[string]interface{}
	json.Unmarshal(alertJSON, &alertMap)
	m.Broadcast("ALERT", alertMap)
}

func (m *Manager) ClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func (m *Manager) OfflineMessageCount() int {
	m.offlineMu.RLock()
	defer m.offlineMu.RUnlock()
	return len(m.offlineQueue)
}

func (m *Manager) GetPendingOfflineMessages(limit int) []OfflineMessage {
	m.offlineMu.RLock()
	defer m.offlineMu.RUnlock()

	pending := make([]OfflineMessage, 0)
	for _, msg := range m.offlineQueue {
		if !msg.Delivered {
			pending = append(pending, msg)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending
}
