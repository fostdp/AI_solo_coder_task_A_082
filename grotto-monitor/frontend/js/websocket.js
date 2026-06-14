class WebSocketManager {
    constructor() {
        this.ws = null;
        this.clientID = this.getOrCreateClientID();
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 15;
        this.reconnectDelay = 3000;
        this.maxReconnectDelay = 30000;
        this.listeners = {};
        this.pendingMessages = this.loadPendingMessages();
        this.acknowledgedMessages = new Set(this.loadAcknowledgedMessages());
        this.isConnecting = false;
        this.manualDisconnect = false;
        this.pingInterval = null;
        this.lastServerTime = null;
    }

    getOrCreateClientID() {
        let clientID = localStorage.getItem('grotto_ws_client_id');
        if (!clientID) {
            clientID = 'client_' + Date.now().toString(36) + '_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem('grotto_ws_client_id', clientID);
        }
        return clientID;
    }

    loadPendingMessages() {
        try {
            const data = localStorage.getItem('grotto_pending_messages');
            return data ? JSON.parse(data) : [];
        } catch (e) {
            console.error('Failed to load pending messages:', e);
            return [];
        }
    }

    savePendingMessages() {
        try {
            const recent = this.pendingMessages.slice(-50);
            localStorage.setItem('grotto_pending_messages', JSON.stringify(recent));
        } catch (e) {
            console.error('Failed to save pending messages:', e);
        }
    }

    loadAcknowledgedMessages() {
        try {
            const data = localStorage.getItem('grotto_acknowledged_messages');
            return data ? JSON.parse(data) : [];
        } catch (e) {
            console.error('Failed to load acknowledged messages:', e);
            return [];
        }
    }

    saveAcknowledgedMessages() {
        try {
            const arr = Array.from(this.acknowledgedMessages).slice(-200);
            localStorage.setItem('grotto_acknowledged_messages', JSON.stringify(arr));
        } catch (e) {
            console.error('Failed to save acknowledged messages:', e);
        }
    }

    connect() {
        if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
            return;
        }

        this.isConnecting = true;
        this.manualDisconnect = false;

        const wsUrl = `ws://localhost:8080/ws/alerts?client_id=${encodeURIComponent(this.clientID)}`;

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log(`WebSocket 连接成功 (Client ID: ${this.clientID})`);
                this.reconnectAttempts = 0;
                this.reconnectDelay = 3000;
                this.isConnecting = false;
                this.updateStatus(true);
                this.startHeartbeat();
                this.emit('connected', { client_id: this.clientID });
                this.flushPendingMessages();
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleMessage(data);
                } catch (e) {
                    console.error('WebSocket 消息解析错误:', e, event.data);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket 错误:', error);
                this.isConnecting = false;
                this.updateStatus(false);
            };

            this.ws.onclose = (event) => {
                console.log(`WebSocket 连接关闭 (code: ${event.code}, reason: ${event.reason})`);
                this.isConnecting = false;
                this.stopHeartbeat();
                this.updateStatus(false);
                this.emit('disconnected');

                if (!this.manualDisconnect) {
                    this.tryReconnect();
                }
            };
        } catch (e) {
            console.error('WebSocket 连接失败:', e);
            this.isConnecting = false;
            this.updateStatus(false);
            this.tryReconnect();
        }
    }

    handleMessage(data) {
        switch (data.type) {
            case 'welcome':
                console.log('收到服务器欢迎消息:', data.data);
                if (data.data?.server_time) {
                    this.lastServerTime = data.data.server_time;
                }
                break;

            case 'alert':
                if (data.message_id && this.acknowledgedMessages.has(data.message_id)) {
                    console.log('消息已确认，跳过重复处理:', data.message_id);
                    return;
                }

                console.log('收到告警消息:', data);
                this.emit('alert', data.data);

                if (data.message_id) {
                    this.acknowledgeMessage(data.message_id);
                }
                break;

            case 'offline_batch':
                console.log(`收到 ${data.data?.count || 0} 条离线消息`);
                this.processOfflineMessages(data.data?.messages || []);
                break;

            case 'ping':
                this.send({ type: 'pong', timestamp: Date.now() });
                break;

            default:
                console.log('收到未知类型消息:', data);
                this.emit(data.type, data.data);
        }
    }

    processOfflineMessages(messages) {
        const filtered = messages.filter(msg => {
            if (msg.id && this.acknowledgedMessages.has(msg.id)) {
                return false;
            }
            const msgAge = Date.now() / 1000 - (msg.timestamp?.seconds || 0);
            if (msgAge > 24 * 60 * 60) {
                return false;
            }
            return true;
        });

        filtered.sort((a, b) => {
            const aTime = a.timestamp?.seconds || 0;
            const bTime = b.timestamp?.seconds || 0;
            return aTime - bTime;
        });

        filtered.forEach((msg, idx) => {
            setTimeout(() => {
                if (msg.type === 'alert') {
                    console.log('处理离线告警:', msg);
                    this.emit('alert', msg.data);
                    if (msg.id) {
                        this.acknowledgeMessage(msg.id);
                    }
                }
                this.emit('offline_message', msg);
            }, idx * 100);
        });

        if (filtered.length > 0) {
            this.showOfflineNotification(filtered.length);
        }
    }

    showOfflineNotification(count) {
        const container = document.getElementById('alert-toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = 'toast';
        toast.style.borderLeft = '4px solid #8b5cf6';
        toast.innerHTML = `
            <div class="toast-header">
                <span>📥 离线消息</span>
            </div>
            <div class="toast-message">
                恢复连接后收到 ${count} 条未读告警消息
            </div>
        `;
        container.appendChild(toast);
        setTimeout(() => toast.remove(), 6000);
    }

    acknowledgeMessage(messageID) {
        this.acknowledgedMessages.add(messageID);
        this.saveAcknowledgedMessages();
        this.send({
            type: 'ack',
            message_id: messageID,
            client_id: this.clientID,
        });
    }

    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify(data));
                return true;
            } catch (e) {
                console.error('发送消息失败:', e);
            }
        }
        return false;
    }

    flushPendingMessages() {
        const remaining = [];
        this.pendingMessages.forEach(msg => {
            if (!this.send(msg)) {
                remaining.push(msg);
            }
        });
        this.pendingMessages = remaining;
        this.savePendingMessages();
    }

    startHeartbeat() {
        this.stopHeartbeat();
        this.pingInterval = setInterval(() => {
            if (this.ws?.readyState === WebSocket.OPEN) {
                this.send({ type: 'ping', client_id: this.clientID, timestamp: Date.now() });
            }
        }, 30000);
    }

    stopHeartbeat() {
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
        }
    }

    tryReconnect() {
        if (this.manualDisconnect) return;
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error(`已达到最大重连次数 (${this.maxReconnectAttempts})，请手动刷新页面重连`);
            this.updateStatus(false, true);
            return;
        }

        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1), this.maxReconnectDelay);
        const jitter = delay * 0.2 * (Math.random() - 0.5);
        const finalDelay = Math.max(1000, delay + jitter);

        console.log(`尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})，${(finalDelay / 1000).toFixed(1)}秒后...`);
        this.updateStatus(false, false, this.reconnectAttempts, this.maxReconnectAttempts, finalDelay);

        setTimeout(() => this.connect(), finalDelay);
    }

    updateStatus(connected, exhausted = false, attempt = 0, maxAttempts = 0, nextDelay = 0) {
        const statusEl = document.getElementById('connection-status');
        if (!statusEl) return;

        const dotEl = statusEl.querySelector('.status-dot');
        const textEl = statusEl.querySelector('.status-text');

        if (connected) {
            statusEl.className = 'status-badge status-connected';
            statusEl.title = `Client ID: ${this.clientID}`;
            if (textEl) textEl.textContent = '已连接';
        } else if (exhausted) {
            statusEl.className = 'status-badge status-error';
            if (textEl) textEl.textContent = '连接失败';
            statusEl.title = '重连次数已达上限，请刷新页面';
        } else if (attempt > 0) {
            statusEl.className = 'status-badge status-reconnecting';
            const delaySec = Math.ceil(nextDelay / 1000);
            if (textEl) textEl.textContent = `重连中 (${attempt}/${maxAttempts}) ${delaySec}s`;
        } else {
            statusEl.className = 'status-badge status-disconnected';
            if (textEl) textEl.textContent = '未连接';
        }
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
        return () => this.off(event, callback);
    }

    off(event, callback) {
        if (this.listeners[event]) {
            this.listeners[event] = this.listeners[event].filter(cb => cb !== callback);
        }
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => {
                try {
                    callback(data);
                } catch (e) {
                    console.error(`Event listener error for ${event}:`, e);
                }
            });
        }
    }

    disconnect() {
        this.manualDisconnect = true;
        this.stopHeartbeat();
        if (this.ws) {
            this.ws.close(1000, 'Manual disconnect');
            this.ws = null;
        }
        this.updateStatus(false);
    }

    resetClientID() {
        localStorage.removeItem('grotto_ws_client_id');
        localStorage.removeItem('grotto_pending_messages');
        localStorage.removeItem('grotto_acknowledged_messages');
        this.clientID = this.getOrCreateClientID();
        this.pendingMessages = [];
        this.acknowledgedMessages = new Set();
        console.log('客户端ID已重置:', this.clientID);
    }

    getConnectionInfo() {
        return {
            client_id: this.clientID,
            connected: this.ws?.readyState === WebSocket.OPEN,
            reconnect_attempts: this.reconnectAttempts,
            pending_messages: this.pendingMessages.length,
            acknowledged_count: this.acknowledgedMessages.size,
            last_server_time: this.lastServerTime,
        };
    }
}

const wsManager = new WebSocketManager();
