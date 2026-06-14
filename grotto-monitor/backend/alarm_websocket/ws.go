package alarm_websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"grotto-monitor/backend/config"
	"grotto-monitor/backend/models"
	"grotto-monitor/backend/repository"
)

type AlertEvent struct {
	SiteID      int       `json:"site_id"`
	SensorID    int       `json:"sensor_id"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	TriggeredAt time.Time `json:"triggered_at"`
}

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
	mu              sync.RWMutex
	messages        map[string][]QueuedMessage
	persistPath     string
	maxPerClient    int
	messageTTL      time.Duration
	cleanupInterval time.Duration
}

func NewOfflineMessageStore(maxPerClient int, messageTTL time.Duration, persistPath string, cleanupInterval time.Duration) *OfflineMessageStore {
	store := &OfflineMessageStore{
		messages:        make(map[string][]QueuedMessage),
		persistPath:     persistPath,
		maxPerClient:    maxPerClient,
		messageTTL:      messageTTL,
		cleanupInterval: cleanupInterval,
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
	ticker := time.NewTicker(s.cleanupInterval)
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
	Clients      map[*Client]bool
	ClientIDMap  map[string]*Client
	Broadcast    chan []byte
	Register     chan *Client
	Unregister   chan *Client
	OfflineStore *OfflineMessageStore
	alertCh      <-chan AlertEvent
	repo         *repository.Repository
	mu           sync.RWMutex
	params       config.AlarmParams
}

func NewHub(alertCh <-chan AlertEvent, repo *repository.Repository, params config.AlarmParams) *Hub {
	offlineStore := NewOfflineMessageStore(
		params.OfflineMaxPerClient,
		time.Duration(params.OfflineTTLHours)*time.Hour,
		params.OfflinePersistPath,
		time.Duration(params.CleanupIntervalMinutes)*time.Minute,
	)
	return &Hub{
		Broadcast:    make(chan []byte, params.BroadcastBufferSize),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Clients:      make(map[*Client]bool),
		ClientIDMap:  make(map[string]*Client),
		OfflineStore: offlineStore,
		alertCh:      alertCh,
		repo:         repo,
		params:       params,
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

		case event := <-h.alertCh:
			alert := models.Alert{
				SiteID:       event.SiteID,
				SensorID:     event.SensorID,
				AlertType:    event.AlertType,
				Severity:     event.Severity,
				Message:      event.Message,
				Value:        event.Value,
				Threshold:    event.Threshold,
				TriggeredAt:  event.TriggeredAt,
				Acknowledged: false,
			}
			if _, err := h.repo.InsertAlert(alert); err != nil {
				log.Printf("Failed to insert alert into DB: %v", err)
			}
			msg := WebSocketMessage{
				Type: "alert",
				Data: event,
			}
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal alert event: %v", err)
				continue
			}
			h.Broadcast <- msgBytes
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
		TTL:       int64(h.params.OfflineTTLHours * 3600),
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
	pongTimeout := time.Duration(c.Hub.params.PongTimeoutSeconds) * time.Second
	c.Conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongTimeout))
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
	pingInterval := time.Duration(c.Hub.params.PingIntervalSeconds) * time.Second
	writeTimeout := time.Duration(c.Hub.params.WriteTimeoutSeconds) * time.Second
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
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
		Send:     make(chan []byte, hub.params.WebSocketBufferSize),
		LastSeen: time.Now(),
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	welcomeMsg, _ := json.Marshal(WebSocketMessage{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id":   clientID,
			"server_time": time.Now().Unix(),
		},
	})
	conn.WriteMessage(websocket.TextMessage, welcomeMsg)

	hub.Register <- client
	go client.WritePump()
	go client.ReadPump()

	log.Printf("New WebSocket connection from %s (client_id: %s)", r.RemoteAddr, clientID)
}

func (h *Hub) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ws/alerts", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(h, w, r)
	}).Methods("GET")
	r.HandleFunc("/api/alerts", h.GetAlerts).Methods("GET")
	r.HandleFunc("/api/alerts/{id}/acknowledge", h.AcknowledgeAlert).Methods("PUT")
}

func (h *Hub) GetAlerts(w http.ResponseWriter, r *http.Request) {
	var filter models.AlertFilter
	q := r.URL.Query()
	if siteIDStr := q.Get("site_id"); siteIDStr != "" {
		siteID, err := strconv.Atoi(siteIDStr)
		if err == nil {
			filter.SiteID = &siteID
		}
	}
	if ackStr := q.Get("acknowledged"); ackStr != "" {
		ack, err := strconv.ParseBool(ackStr)
		if err == nil {
			filter.Acknowledged = &ack
		}
	}

	alerts, err := h.repo.GetAlerts(filter)
	if err != nil {
		log.Printf("Failed to get alerts: %v", err)
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}
	if alerts == nil {
		alerts = []models.Alert{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (h *Hub) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.AcknowledgeAlert(id); err != nil {
		log.Printf("Failed to acknowledge alert %d: %v", id, err)
		http.Error(w, "Failed to acknowledge alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           id,
		"acknowledged": true,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
