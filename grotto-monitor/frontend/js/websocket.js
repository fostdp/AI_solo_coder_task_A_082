class WebSocketManager {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 3000;
        this.listeners = {};
    }

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `ws://localhost:8080/ws/alerts`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket 连接成功');
                this.reconnectAttempts = 0;
                this.updateStatus(true);
                this.emit('connected');
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    if (data.type === 'alert') {
                        this.emit('alert', data.alert);
                    }
                } catch (e) {
                    console.error('WebSocket 消息解析错误:', e);
                }
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket 错误:', error);
                this.updateStatus(false);
            };

            this.ws.onclose = () => {
                console.log('WebSocket 连接关闭');
                this.updateStatus(false);
                this.tryReconnect();
            };
        } catch (e) {
            console.error('WebSocket 连接失败:', e);
            this.updateStatus(false);
            this.tryReconnect();
        }
    }

    tryReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
            setTimeout(() => this.connect(), this.reconnectDelay);
        } else {
            console.error('已达到最大重连次数');
        }
    }

    updateStatus(connected) {
        const statusEl = document.getElementById('connection-status');
        if (statusEl) {
            if (connected) {
                statusEl.className = 'status-badge status-connected';
                statusEl.querySelector('.status-text').textContent = '已连接';
            } else {
                statusEl.className = 'status-badge status-disconnected';
                statusEl.querySelector('.status-text').textContent = '未连接';
            }
        }
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => callback(data));
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }
}

const wsManager = new WebSocketManager();
