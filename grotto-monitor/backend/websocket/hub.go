package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type QueuedMessage struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	TTL       int64       `json:"ttl"`
}

type OfflineMessageStore struct {
	mu        sync.RWMutex
	messages  map[string][]QueuedMessage
	persistPath string
	maxPerClient int
	messageTTL  time.Duration
}

func NewOfflineMessageStore(persistPath string) *OfflineMessageStore {
	store := &OfflineMessageStore{
		messages:     make(map[string][]QueuedMessage),
		persistPath:  persistPath,
		maxPerClient: 100,
		messageTTL:   24 * time.Hour,
	}
	store.load()
	go store.cleanupLoop()
	return store
}

func (s *OfflineMessageStore) load() {
	if s.persistPath == "" {
		return
	}
	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to load offline messages: %v", err)
		}
		return
	}
	var loaded map[string][]QueuedMessage
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("Failed to parse offline messages: %v", err)
		return
	}
	now := time.Now()
	for clientID, msgs := range loaded {
		var valid []QueuedMessage
		for _, msg := range msgs {
			if now.Sub(msg.Timestamp) < s.messageTTL {
				valid = append(valid, msg)
			}
		}
		if len(valid) > 0 {
			s.messages[clientID] = valid
		}
	}
	log.Printf("Loaded %d client offline message queues", len(s.messages))
}

func (s *OfflineMessageStore) save() {
	if s.persistPath == "" {
		return
	}
	s.mu.RLock()
	data, err := json.Marshal(s.messages)
	s.mu.RUnlock()
	if err != nil {
		log.Printf("Failed to marshal offline messages: %v", err)
		return
	}
	dir := filepath.Dir(s.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create persist directory: %v", err)
		return
	}
	if err := os.WriteFile(s.persistPath, data, 0644); err != nil {
		log.Printf("Failed to save offline messages: %v", err)
	}
}

func (s *OfflineMessageStore) Enqueue(clientID string, msg QueuedMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messages[clientID]) >= s.maxPerClient {
		s.messages[clientID] = s.messages[clientID][1:]
	}
	s.messages[clientID] = append(s.messages[clientID], msg)
}

func (s *OfflineMessageStore) DequeueAll(clientID string) []QueuedMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs := s.messages[clientID]
	delete(s.messages, clientID)
	return msgs
}

func (s *OfflineMessageStore) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
		s.save()
	}
}

func (s *OfflineMessageStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for clientID, msgs := range s.messages {
		var valid []QueuedMessage
		for _, msg := range msgs {
			if now.Sub(msg.Timestamp) < s.messageTTL {
				valid = append(valid, msg)
			}
		}
		if len(valid) == 0 {
			delete(s.messages, clientID)
		} else if len(valid) != len(msgs) {
			s.messages[clientID] = valid
		}
	}
}

type Client struct {
	ID       string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	LastSeen time.Time
}

type Hub struct {
	Clients        map[*Client]bool
	ClientIDMap    map[string]*Client
	Broadcast      chan []byte
	Register       chan *Client
	Unregister     chan *Client
	OfflineStore   *OfflineMessageStore
	mu             sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:    make(chan []byte, 512),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Clients:      make(map[*Client]bool),
		ClientIDMap:  make(map[string]*Client),
		OfflineStore: NewOfflineMessageStore("data/offline_messages.json"),
	}
}

type WebSocketMessage struct {
	Type      string      `json:"type"`
	ClientID  string      `json:"client_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	MessageID string      `json:"message_id,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.ClientIDMap[client.ID] = client
			h.mu.Unlock()

			log.Printf("Client registered: %s (total: %d)", client.ID, len(h.Clients))

			go h.sendOfflineMessages(client)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				if existing, exists := h.ClientIDMap[client.ID]; exists && existing == client {
					delete(h.ClientIDMap, client.ID)
				}
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %s (remaining: %d)", client.ID, len(h.Clients))

		case message := <-h.Broadcast:
			h.handleBroadcast(message)
		}
	}
}

func (h *Hub) handleBroadcast(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var parsed WebSocketMessage
	if err := json.Unmarshal(message, &parsed); err != nil {
		for client := range h.Clients {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(h.Clients, client)
				if existing, exists := h.ClientIDMap[client.ID]; exists && existing == client {
					delete(h.ClientIDMap, client.ID)
				}
			}
		}
		return
	}

	msgID := uuid.New().String()
	queuedMsg := QueuedMessage{
		ID:        msgID,
		Type:      parsed.Type,
		Data:      parsed.Data,
		Timestamp: time.Now(),
		TTL:       int64(24 * time.Hour / time.Second),
	}

	queuedBytes, _ := json.Marshal(WebSocketMessage{
		Type:      parsed.Type,
		Data:      parsed.Data,
		MessageID: msgID,
		Timestamp: queuedMsg.Timestamp.Unix(),
	})

	for client := range h.Clients {
		select {
		case client.Send <- queuedBytes:
		default:
			h.OfflineStore.Enqueue(client.ID, queuedMsg)
			close(client.Send)
			delete(h.Clients, client)
			if existing, exists := h.ClientIDMap[client.ID]; exists && existing == client {
				delete(h.ClientIDMap, client.ID)
			}
		}
	}

	h.OfflineStore.save()
}

func (h *Hub) sendOfflineMessages(client *Client) {
	messages := h.OfflineStore.DequeueAll(client.ID)
	if len(messages) == 0 {
		return
	}

	log.Printf("Sending %d offline messages to client %s", len(messages), client.ID)

	batchMsg := WebSocketMessage{
		Type: "offline_batch",
		Data: map[string]interface{}{
			"count":    len(messages),
			"messages": messages,
		},
	}

	batchBytes, err := json.Marshal(batchMsg)
	if err != nil {
		log.Printf("Failed to marshal offline batch: %v", err)
		return
	}

	select {
	case client.Send <- batchBytes:
	default:
		for _, msg := range messages {
			h.OfflineStore.Enqueue(client.ID, msg)
		}
	}

	h.OfflineStore.save()
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		c.LastSeen = time.Now()
		return nil
	})
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error [client=%s]: %v", c.ID, err)
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "ack" && msg.MessageID != "" {
			log.Printf("Client %s acknowledged message %s", c.ID, msg.MessageID)
		}

		c.LastSeen = time.Now()
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
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
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = uuid.New().String()
	}

	client := &Client{
		ID:       clientID,
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, 512),
		LastSeen: time.Now(),
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	welcomeMsg, _ := json.Marshal(WebSocketMessage{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id": clientID,
			"server_time": time.Now().Unix(),
		},
	})
	conn.WriteMessage(websocket.TextMessage, welcomeMsg)

	hub.Register <- client
	go client.WritePump()
	go client.ReadPump()

	log.Printf("New WebSocket connection from %s (client_id: %s)", r.RemoteAddr, clientID)
}

func BroadcastAlert(hub *Hub, alertData interface{}) {
	msg := WebSocketMessage{
		Type: "alert",
		Data: alertData,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal alert: %v", err)
		return
	}
	hub.Broadcast <- msgBytes
}
