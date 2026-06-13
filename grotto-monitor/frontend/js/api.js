const API_BASE_URL = 'http://localhost:8080/api';

const api = {
    async request(endpoint, options = {}) {
        const url = `${API_BASE_URL}${endpoint}`;
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
            },
        };
        const finalOptions = { ...defaultOptions, ...options };
        
        try {
            const response = await fetch(url, finalOptions);
            const result = await response.json();
            if (result.success) {
                return result.data;
            } else {
                throw new Error(result.error || '请求失败');
            }
        } catch (error) {
            console.error(`API Error [${endpoint}]:`, error);
            throw error;
        }
    },

    getSites() {
        return this.request('/sites');
    },

    getSite(id) {
        return this.request(`/sites/${id}`);
    },

    getSensorsBySite(siteId) {
        return this.request(`/sites/${siteId}/sensors`);
    },

    getMonitoringData(siteId, hours = 24) {
        return this.request(`/sites/${siteId}/data?hours=${hours}`);
    },

    getHourlyData(sensorId, days = 30) {
        return this.request(`/sensors/${sensorId}/hourly?days=${days}`);
    },

    getWeatheringRates(siteId, days = 365) {
        return this.request(`/sites/${siteId}/weathering-rates?days=${days}`);
    },

    insertMonitoringData(data) {
        return this.request('/data', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    getAlerts(siteId = null, acknowledged = null) {
        let query = '';
        const params = [];
        if (siteId !== null) params.push(`site_id=${siteId}`);
        if (acknowledged !== null) params.push(`acknowledged=${acknowledged}`);
        if (params.length > 0) query = `?${params.join('&')}`;
        return this.request(`/alerts${query}`);
    },

    acknowledgeAlert(alertId) {
        return this.request(`/alerts/${alertId}/acknowledge`, {
            method: 'PUT',
        });
    },

    getMaterials() {
        return this.request('/materials');
    },

    predictWeatheringRate(data) {
        return this.request('/predict', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    predictBatchRates(data) {
        return this.request('/predict/batch', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    optimizeMaterials(data) {
        return this.request('/topsis', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },
};
