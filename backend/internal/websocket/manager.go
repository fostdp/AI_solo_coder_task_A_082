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

type Manager struct {
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
	broadcast chan models.WebSocketMessage
}

var (
	wsManager *Manager
	once      sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		wsManager = &Manager{
			clients:   make(map[*websocket.Conn]bool),
			broadcast: make(chan models.WebSocketMessage, 256),
		}
		go wsManager.run()
	})
	return wsManager
}

func (m *Manager) run() {
	for msg := range m.broadcast {
		m.mu.RLock()
		for client := range m.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(m.clients, client)
			}
		}
		m.mu.RUnlock()
	}
}

func (m *Manager) HandleConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	m.mu.Lock()
	m.clients[conn] = true
	connCount := len(m.clients)
	m.mu.Unlock()

	log.Printf("WebSocket client connected, total: %d", connCount)

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
			client := conn
			m.mu.RUnlock()
			if err := client.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
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
