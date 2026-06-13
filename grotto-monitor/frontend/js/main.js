let viewer3D = null;
let currentSite = null;
let sites = [];
let sensors = [];
let alerts = [];
let hourlyChart = null;
let weatheringChart = null;
let unacknowledgedAlerts = 0;

document.addEventListener('DOMContentLoaded', () => {
    init();
});

async function init() {
    viewer3D = new Grotto3DViewer('three-container');
    viewer3D.onSensorClick = handleSensorClick;

    setupEventListeners();
    
    try {
        await loadSites();
        setupWebSocket();
        loadAlerts();
    } catch (error) {
        console.error('初始化失败:', error);
    }
}

function setupEventListeners() {
    document.getElementById('btn-prediction').addEventListener('click', () => switchView('prediction'));
    document.getElementById('btn-topsis').addEventListener('click', () => switchView('topsis'));
    document.getElementById('btn-alerts').addEventListener('click', () => {
        switchView('alerts');
        loadAlerts();
    });

    document.getElementById('btn-do-predict').addEventListener('click', doPrediction);
    document.getElementById('btn-batch-predict').addEventListener('click', doBatchPrediction);
    document.getElementById('btn-do-topsis').addEventListener('click', doTOPSIS);

    document.getElementById('prediction-temp').addEventListener('input', (e) => {
        document.getElementById('temp-value').textContent = `${e.target.value}°C`;
    });
    document.getElementById('prediction-humidity').addEventListener('input', (e) => {
        document.getElementById('humidity-value').textContent = `${e.target.value}%`;
    });

    ['penetration', 'breathability', 'weathering', 'compatibility', 'cost', 'durability'].forEach(attr => {
        document.getElementById(`weight-${attr}`).addEventListener('input', (e) => {
            document.getElementById(`weight-${attr}-value`).textContent = parseFloat(e.target.value).toFixed(2);
        });
    });

    document.getElementById('filter-unacknowledged').addEventListener('change', loadAlerts);
    document.getElementById('filter-site').addEventListener('change', loadAlerts);

    document.getElementById('alert-count').addEventListener('click', () => {
        switchView('alerts');
        loadAlerts();
    });

    window.closeModal = closeModal;
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeModal();
    });
    document.getElementById('sensor-modal').addEventListener('click', (e) => {
        if (e.target.id === 'sensor-modal') closeModal();
    });
}

function setupWebSocket() {
    wsManager.on('alert', (alert) => {
        showAlertToast(alert);
        updateAlertCount();
    });
    wsManager.connect();
}

async function loadSites() {
    try {
        sites = await api.getSites();
        renderSiteList();
        populateSiteSelects();
        
        if (sites.length > 0) {
            selectSite(sites[0]);
        }
    } catch (error) {
        console.error('加载石窟列表失败:', error);
        document.getElementById('site-list').innerHTML = `
            <p class="empty-hint">无法加载数据，请确保后端服务已启动</p>
        `;
    }
}

function renderSiteList() {
    const listEl = document.getElementById('site-list');
    listEl.innerHTML = sites.map(site => `
        <div class="site-item" data-id="${site.id}" onclick="selectSiteById(${site.id})">
            <div class="site-item-name">${site.name}</div>
            <div class="site-item-location">${site.location}</div>
            <span class="site-item-rock">${site.rock_type}</span>
        </div>
    `).join('');
}

function populateSiteSelects() {
    const options = sites.map(site => `<option value="${site.id}">${site.name}</option>`).join('');
    document.getElementById('prediction-site').innerHTML = options;
    document.getElementById('filter-site').innerHTML += options;
}

window.selectSiteById = async function(siteId) {
    const site = sites.find(s => s.id === siteId);
    if (site) {
        selectSite(site);
    }
};

async function selectSite(site) {
    currentSite = site;
    
    document.querySelectorAll('.site-item').forEach(el => {
        el.classList.toggle('active', parseInt(el.dataset.id) === site.id);
    });

    updateCurrentSiteInfo(site);
    viewer3D.setSite(site);

    try {
        sensors = await api.getSensorsBySite(site.id);
        viewer3D.setSensors(sensors);
        viewer3D.updateHeatmap(generateHeatmapData(sensors));
    } catch (error) {
        console.error('加载传感器数据失败:', error);
    }

    switchView('3d');
}

function updateCurrentSiteInfo(site) {
    const infoEl = document.getElementById('current-site-info');
    infoEl.innerHTML = `
        <div class="info-row">
            <span class="info-label">名称</span>
            <span class="info-value">${site.name}</span>
        </div>
        <div class="info-row">
            <span class="info-label">位置</span>
            <span class="info-value">${site.location}</span>
        </div>
        <div class="info-row">
            <span class="info-label">岩石类型</span>
            <span class="info-value">${site.rock_type}</span>
        </div>
        <div class="info-row">
            <span class="info-label">坐标</span>
            <span class="info-value">${site.latitude.toFixed(4)}, ${site.longitude.toFixed(4)}</span>
        </div>
    `;
}

function generateHeatmapData(sensors) {
    return sensors.map(sensor => ({
        x: sensor.position_x - 20,
        y: sensor.position_z - 20,
        intensity: sensor.sensor_type === '表面硬度' ? 0.3 + Math.random() * 0.5 : 0.2 + Math.random() * 0.3
    }));
}

function switchView(viewName) {
    document.querySelectorAll('.view').forEach(view => view.classList.remove('active'));
    document.querySelectorAll('.tool-btn').forEach(btn => btn.classList.remove('active'));
    
    document.getElementById(`view-${viewName}`).classList.add('active');
    
    if (viewName !== '3d') {
        document.getElementById(`btn-${viewName}`)?.classList.add('active');
    }
}

function handleSensorClick(sensor) {
    showSensorModal(sensor);
}

async function showSensorModal(sensor) {
    const modal = document.getElementById('sensor-modal');
    const titleEl = document.getElementById('modal-title');
    const infoEl = document.getElementById('modal-sensor-info');

    titleEl.textContent = `${sensor.sensor_code} - ${sensor.sensor_type}`;
    
    const site = sites.find(s => s.id === sensor.site_id);
    const unit = getSensorUnit(sensor.sensor_type);
    
    infoEl.innerHTML = `
        <h4>监测点基本信息</h4>
        <div class="sensor-info-grid">
            <div class="sensor-info-item">
                <div class="label">石窟</div>
                <div class="value">${site?.name || '-'}</div>
            </div>
            <div class="sensor-info-item">
                <div class="label">类型</div>
                <div class="value">${sensor.sensor_type}</div>
            </div>
            <div class="sensor-info-item">
                <div class="label">状态</div>
                <div class="value">${sensor.status === 'active' ? '正常' : '离线'}</div>
            </div>
            <div class="sensor-info-item">
                <div class="label">位置</div>
                <div class="value">(${sensor.position_x.toFixed(1)}, ${sensor.position_y.toFixed(1)}, ${sensor.position_z.toFixed(1)})</div>
            </div>
            <div class="sensor-info-item">
                <div class="label">基准值</div>
                <div class="value">${sensor.baseline_value > 0 ? sensor.baseline_value.toFixed(2) + unit : '-'}</div>
            </div>
            <div class="sensor-info-item">
                <div class="label">安装时间</div>
                <div class="value">${new Date(sensor.installed_at).toLocaleDateString()}</div>
            </div>
        </div>
    `;

    modal.classList.remove('hidden');
    
    await loadSensorCharts(sensor);
}

function getSensorUnit(type) {
    switch (type) {
        case '温度': return '°C';
        case '湿度': return '%';
        case '表面硬度': return ' H';
        case '裂隙宽度': return 'mm';
        default: return '';
    }
}

async function loadSensorCharts(sensor) {
    try {
        const [hourlyData, weatheringData] = await Promise.all([
            api.getHourlyData(sensor.id, 30),
            api.getWeatheringRates(sensor.site_id, 365)
        ]);

        renderHourlyChart(hourlyData, sensor);
        renderWeatheringChart(weatheringData);
    } catch (error) {
        console.error('加载图表数据失败:', error);
    }
}

function renderHourlyChart(data, sensor) {
    const ctx = document.getElementById('chart-hourly').getContext('2d');
    
    if (hourlyChart) {
        hourlyChart.destroy();
    }

    const unit = getSensorUnit(sensor.sensor_type);
    const labels = data.map(d => new Date(d.bucket).toLocaleDateString());
    const avgValues = data.map(d => d.avg_value);
    const minValues = data.map(d => d.min_value);
    const maxValues = data.map(d => d.max_value);

    hourlyChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [
                {
                    label: '平均值',
                    data: avgValues,
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4,
                },
                {
                    label: '最高值',
                    data: maxValues,
                    borderColor: '#ef4444',
                    borderDash: [5, 5],
                    pointRadius: 0,
                    tension: 0.4,
                },
                {
                    label: '最低值',
                    data: minValues,
                    borderColor: '#22c55e',
                    borderDash: [5, 5],
                    pointRadius: 0,
                    tension: 0.4,
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    labels: { color: '#94a3b8' }
                }
            },
            scales: {
                x: {
                    ticks: { color: '#94a3b8', maxTicksLimit: 10 },
                    grid: { color: 'rgba(100, 116, 139, 0.2)' }
                },
                y: {
                    ticks: { color: '#94a3b8', callback: v => v + unit },
                    grid: { color: 'rgba(100, 116, 139, 0.2)' }
                }
            }
        }
    });
}

function renderWeatheringChart(data) {
    const ctx = document.getElementById('chart-weathering').getContext('2d');
    
    if (weatheringChart) {
        weatheringChart.destroy();
    }

    const labels = data.map(d => new Date(d.time).toLocaleDateString());
    const rates = data.map(d => (d.rate * 100).toFixed(4));

    weatheringChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: '风化速率 (%)',
                data: rates,
                backgroundColor: rates.map(r => 
                    parseFloat(r) > 0.8 ? 'rgba(239, 68, 68, 0.7)' :
                    parseFloat(r) > 0.5 ? 'rgba(249, 115, 22, 0.7)' :
                    parseFloat(r) > 0.3 ? 'rgba(245, 158, 11, 0.7)' :
                    'rgba(34, 197, 94, 0.7)'
                ),
                borderWidth: 0,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    labels: { color: '#94a3b8' }
                }
            },
            scales: {
                x: {
                    ticks: { color: '#94a3b8', maxTicksLimit: 12 },
                    grid: { color: 'rgba(100, 116, 139, 0.2)' }
                },
                y: {
                    ticks: { color: '#94a3b8', callback: v => v + '%' },
                    grid: { color: 'rgba(100, 116, 139, 0.2)' }
                }
            }
        }
    });
}

function closeModal() {
    document.getElementById('sensor-modal').classList.add('hidden');
    if (hourlyChart) {
        hourlyChart.destroy();
        hourlyChart = null;
    }
    if (weatheringChart) {
        weatheringChart.destroy();
        weatheringChart = null;
    }
}

async function doPrediction() {
    const siteId = parseInt(document.getElementById('prediction-site').value);
    const temperature = parseFloat(document.getElementById('prediction-temp').value);
    const humidity = parseFloat(document.getElementById('prediction-humidity').value);

    try {
        const result = await api.predictWeatheringRate({
            site_id: siteId,
            temperature: temperature,
            humidity: humidity
        });

        renderPredictionResult(result);
    } catch (error) {
        document.getElementById('prediction-result').innerHTML = `
            <p class="empty-hint">预测失败: ${error.message}</p>
        `;
    }
}

function renderPredictionResult(result) {
    const riskClass = result.risk_level === '低风险' ? 'low' :
                      result.risk_level === '中等风险' ? 'medium' :
                      result.risk_level === '高风险' ? 'high' : 'extreme';

    document.getElementById('prediction-result').innerHTML = `
        <div class="result-card risk-${riskClass}">
            <div class="result-rate">${(result.predicted_rate * 100).toFixed(2)}%</div>
            <span class="result-risk ${riskClass}">${result.risk_level}</span>
            <div class="result-confidence">置信度: ${(result.confidence * 100).toFixed(1)}%</div>
            <div class="result-recommendation">${result.recommendation}</div>
        </div>
    `;
}

async function doBatchPrediction() {
    const siteId = parseInt(document.getElementById('prediction-site').value);
    
    const combinations = [];
    for (let temp = -5; temp <= 45; temp += 10) {
        for (let hum = 20; hum <= 90; hum += 20) {
            combinations.push({ site_id: siteId, temperature: temp, humidity: hum });
        }
    }

    try {
        const result = await api.predictBatchRates({
            site_id: siteId,
            combinations: combinations
        });

        renderBatchResult(result.predictions, combinations);
    } catch (error) {
        document.getElementById('batch-result').innerHTML = `
            <p class="empty-hint">批量预测失败: ${error.message}</p>
        `;
    }
}

function renderBatchResult(predictions, combinations) {
    const container = document.getElementById('batch-result');
    container.innerHTML = predictions.map((pred, idx) => {
        const comb = combinations[idx];
        const riskClass = pred.risk_level === '低风险' ? 'low' :
                          pred.risk_level === '中等风险' ? 'medium' :
                          pred.risk_level === '高风险' ? 'high' : 'extreme';
        
        return `
            <div class="batch-item">
                <div class="batch-item-params">${comb.temperature}°C / ${comb.humidity}%</div>
                <div class="batch-item-rate">${(pred.predicted_rate * 100).toFixed(2)}%</div>
                <div class="result-risk ${riskClass}" style="display:inline-block;padding:2px 6px;border-radius:8px;font-size:10px;margin-top:4px;">${pred.risk_level}</div>
            </div>
        `;
    }).join('');
}

async function doTOPSIS() {
    const rockType = document.getElementById('topsis-rocktype').value;
    const priorities = {
        penetration_depth: parseFloat(document.getElementById('weight-penetration').value),
        breathability: parseFloat(document.getElementById('weight-breathability').value),
        weathering_resistance: parseFloat(document.getElementById('weight-weathering').value),
        compatibility: parseFloat(document.getElementById('weight-compatibility').value),
        cost: parseFloat(document.getElementById('weight-cost').value),
        durability_years: parseFloat(document.getElementById('weight-durability').value),
    };

    const totalWeight = Object.values(priorities).reduce((a, b) => a + b, 0);
    for (const key in priorities) {
        priorities[key] = priorities[key] / totalWeight;
    }

    try {
        const result = await api.optimizeMaterials({
            site_id: currentSite?.id || 1,
            rock_type: rockType,
            priorities: priorities,
        });

        renderTOPSISResult(result, rockType);
    } catch (error) {
        document.getElementById('topsis-result').innerHTML = `
            <p class="empty-hint">优化分析失败: ${error.message}</p>
        `;
    }
}

function renderTOPSISResult(result, rockType) {
    const container = document.getElementById('topsis-result');
    const materials = result.results;

    let html = `
        <h4 style="margin-bottom: 16px; color: #f1f5f9;">岩石类型: ${rockType} - 保护材料推荐排名</h4>
        <table>
            <thead>
                <tr>
                    <th>排名</th>
                    <th>材料名称</th>
                    <th>TOPSIS 得分</th>
                    <th>推荐度</th>
                </tr>
            </thead>
            <tbody>
    `;

    materials.forEach((m, idx) => {
        const rankClass = idx === 0 ? 'rank-1' : idx === 1 ? 'rank-2' : idx === 2 ? 'rank-3' : '';
        const medal = idx === 0 ? '🥇' : idx === 1 ? '🥈' : idx === 2 ? '🥉' : `${idx + 1}`;
        const scorePercent = (m.topsis_score * 100).toFixed(2);

        html += `
            <tr class="${rankClass}">
                <td>${medal}</td>
                <td>${m.material_name}</td>
                <td>${scorePercent}%</td>
                <td>
                    <div style="width: 100px; height: 8px; background: rgba(100, 116, 139, 0.2); border-radius: 4px; overflow: hidden;">
                        <div style="width: ${scorePercent}%; height: 100%; background: linear-gradient(90deg, #3b82f6, #8b5cf6);"></div>
                    </div>
                </td>
            </tr>
        `;
    });

    html += `
            </tbody>
        </table>
        <div style="margin-top: 20px; padding: 16px; background: rgba(59, 130, 246, 0.1); border-radius: 8px; border-left: 4px solid #3b82f6;">
            <h4 style="color: #93c5fd; margin-bottom: 8px;">💡 推荐方案</h4>
            <p style="color: #cbd5e1; font-size: 13px; line-height: 1.6;">
                基于 TOPSIS 多属性决策分析，针对 ${rockType} 类型岩石，
                <strong style="color: #f59e0b;">${materials[0].material_name}</strong> 得分最高 (${(materials[0].topsis_score * 100).toFixed(2)}%)，
                是最优保护材料选择。建议结合实际工况进行小面积试验后再大规模应用。
            </p>
        </div>
    `;

    container.innerHTML = html;
}

async function loadAlerts() {
    const unacknowledgedOnly = document.getElementById('filter-unacknowledged')?.checked;
    const siteFilter = document.getElementById('filter-site')?.value;

    try {
        alerts = await api.getAlerts(
            siteFilter ? parseInt(siteFilter) : null,
            unacknowledgedOnly ? false : null
        );
        renderAlerts();
        updateAlertCount();
    } catch (error) {
        console.error('加载告警失败:', error);
    }
}

function renderAlerts() {
    const container = document.getElementById('alerts-list');
    
    if (alerts.length === 0) {
        container.innerHTML = '<p class="empty-hint">暂无告警信息</p>';
        return;
    }

    container.innerHTML = alerts.map(alert => {
        const site = sites.find(s => s.id === alert.site_id);
        const alertClass = alert.alert_type.includes('裂隙') ? 'crack' : 'hardness';
        const ackClass = alert.acknowledged ? 'acknowledged' : '';

        return `
            <div class="alert-item ${ackClass}" data-id="${alert.id}">
                <div class="alert-content">
                    <div class="alert-header">
                        <span class="alert-type ${alertClass}">${alert.alert_type}</span>
                        <span class="alert-severity">${alert.severity === 'warning' ? '⚠️ 警告' : '🔴 严重'}</span>
                        <span style="color: #64748b; font-size: 12px;">${site?.name || '未知石窟'}</span>
                    </div>
                    <div class="alert-message">${alert.message}</div>
                    <div class="alert-meta">
                        <span>当前值: ${alert.value.toFixed(3)}</span>
                        <span>阈值: ${alert.threshold.toFixed(3)}</span>
                        <span>触发时间: ${new Date(alert.triggered_at).toLocaleString()}</span>
                        ${alert.acknowledged ? '<span>✅ 已确认</span>' : ''}
                    </div>
                </div>
                ${!alert.acknowledged ? `
                <div class="alert-actions">
                    <button onclick="acknowledgeAlert(${alert.id})">确认</button>
                </div>
                ` : ''}
            </div>
        `;
    }).join('');
}

window.acknowledgeAlert = async function(alertId) {
    try {
        await api.acknowledgeAlert(alertId);
        await loadAlerts();
    } catch (error) {
        console.error('确认告警失败:', error);
    }
};

function showAlertToast(alert) {
    const container = document.getElementById('alert-toast-container');
    const site = sites.find(s => s.id === alert.site_id);

    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.innerHTML = `
        <div class="toast-header">
            <span>🚨 新告警</span>
            <span style="margin-left: auto; font-size: 12px; opacity: 0.8;">${site?.name || ''}</span>
        </div>
        <div class="toast-message">${alert.message}</div>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        toast.remove();
    }, 5000);
}

function updateAlertCount() {
    unacknowledgedAlerts = alerts.filter(a => !a.acknowledged).length;
    const badge = document.getElementById('alert-count');
    
    if (unacknowledgedAlerts > 0) {
        badge.classList.remove('hidden');
        badge.querySelector('.alert-number').textContent = unacknowledgedAlerts;
    } else {
        badge.classList.add('hidden');
    }
}