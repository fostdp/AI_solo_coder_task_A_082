let viewer3D = null;
let panel = null;
let currentSite = null;
let sites = [];
let sensors = [];

document.addEventListener('DOMContentLoaded', () => {
    init();
});

async function init() {
    viewer3D = new Grotto3DViewer('three-container');
    panel = new WeatheringPanel();
    viewer3D.onSensorClick = (sensor) => panel.showSensorModal(sensor);

    setupEventListeners();

    try {
        await loadSites();
        setupWebSocket();
        panel.loadAlerts();
    } catch (error) {
        console.error('初始化失败:', error);
    }
}

function setupEventListeners() {
    document.getElementById('btn-prediction').addEventListener('click', () => switchView('prediction'));
    document.getElementById('btn-topsis').addEventListener('click', () => switchView('topsis'));
    document.getElementById('btn-alerts').addEventListener('click', () => {
        switchView('alerts');
        panel.loadAlerts();
    });

    document.getElementById('btn-do-predict').addEventListener('click', () => panel.doPrediction());
    document.getElementById('btn-batch-predict').addEventListener('click', () => panel.doBatchPrediction());
    document.getElementById('btn-do-topsis').addEventListener('click', () => panel.doTOPSIS());

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

    document.getElementById('filter-unacknowledged').addEventListener('change', () => panel.loadAlerts());
    document.getElementById('filter-site').addEventListener('change', () => panel.loadAlerts());

    document.getElementById('alert-count').addEventListener('click', () => {
        switchView('alerts');
        panel.loadAlerts();
    });

    window.closeModal = () => panel.closeModal();
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') panel.closeModal();
    });
    document.getElementById('sensor-modal').addEventListener('click', (e) => {
        if (e.target.id === 'sensor-modal') panel.closeModal();
    });
}

function setupWebSocket() {
    wsManager.on('alert', (alert) => {
        panel.showAlertToast(alert);
        panel.updateAlertCount();
    });
    wsManager.on('offline_message', (msg) => {
        if (msg.type === 'alert') {
            panel.showAlertToast(msg.data);
        }
    });
    wsManager.connect();
}

async function loadSites() {
    try {
        sites = await api.getSites();
        panel.setSites(sites);
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
    if (site) selectSite(site);
};

async function selectSite(site) {
    currentSite = site;
    panel.setCurrentSite(site);

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
