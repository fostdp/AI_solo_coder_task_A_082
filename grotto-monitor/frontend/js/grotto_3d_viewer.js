class Grotto3DViewer {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.sensors = [];
        this.sensorMeshes = [];
        this.crackLines = [];
        this.grottoLODs = [];
        this.currentLODLevel = 0;
        this.heatmapOverlay = null;
        this.showHeatmap = true;
        this.showCracks = true;
        this.showSensors = true;
        this.currentSite = null;
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.onSensorClick = null;
        this.zoom = 1;
        this.rotationX = 0;
        this.rotationY = 0;
        this.target = new THREE.Vector3(0, 0, 0);
        this.frameCount = 0;
        this.lodCheckInterval = 30;
        this.lowPolyDistance = 60;
        this.mediumPolyDistance = 35;
        this.textureCache = new Map();
        this.loadingStage = 0;
        this.isLoading = false;
        this.animationId = null;

        this.init();
    }

    init() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a0a0f);
        this.scene.fog = new THREE.Fog(0x0a0a0f, 60, 160);

        this.camera = new THREE.PerspectiveCamera(55, width / height, 0.1, 500);
        this.camera.position.set(30, 20, 30);
        this.camera.lookAt(0, 0, 0);

        const canvas = document.createElement('canvas');
        canvas.width = 2;
        canvas.height = 2;
        const ctx = canvas.getContext('2d');
        ctx.fillStyle = '#d4a574';
        ctx.fillRect(0, 0, 2, 2);
        this.defaultTexture = new THREE.CanvasTexture(canvas);

        this.renderer = new THREE.WebGLRenderer({
            antialias: this.isHighEndDevice(),
            alpha: true,
            powerPreference: 'high-performance'
        });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, this.isHighEndDevice() ? 2 : 1.5));
        this.renderer.shadowMap.enabled = this.isHighEndDevice();
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.renderer.outputColorSpace = THREE.SRGBColorSpace;
        this.container.appendChild(this.renderer.domElement);

        this.setupLighting();
        this.setupControls();
        this.setupEventListeners();
        this.animate();

        window.addEventListener('resize', () => this.onResize());
    }

    isHighEndDevice() {
        const cores = navigator.hardwareConcurrency || 4;
        const memory = navigator.deviceMemory || 4;
        return cores >= 6 && memory >= 4;
    }

    setupLighting() {
        const ambientIntensity = this.isHighEndDevice() ? 0.5 : 0.6;
        const ambientLight = new THREE.AmbientLight(0x404050, ambientIntensity);
        this.scene.add(ambientLight);

        const mainLight = new THREE.DirectionalLight(0xffffff, this.isHighEndDevice() ? 1.2 : 1.0);
        mainLight.position.set(50, 50, 30);
        mainLight.castShadow = this.isHighEndDevice();
        if (this.isHighEndDevice()) {
            mainLight.shadow.mapSize.width = 1024;
            mainLight.shadow.mapSize.height = 1024;
            mainLight.shadow.camera.near = 0.5;
            mainLight.shadow.camera.far = 200;
            mainLight.shadow.camera.left = -50;
            mainLight.shadow.camera.right = 50;
            mainLight.shadow.camera.top = 50;
            mainLight.shadow.camera.bottom = -50;
        }
        this.scene.add(mainLight);

        const fillLight = new THREE.DirectionalLight(0x4a90d9, 0.25);
        fillLight.position.set(-30, 20, -30);
        this.scene.add(fillLight);

        if (this.isHighEndDevice()) {
            const rimLight = new THREE.DirectionalLight(0xff9500, 0.35);
            rimLight.position.set(-20, 10, 40);
            this.scene.add(rimLight);
        }
    }

    setupControls() {
        let isMouseDown = false;
        let mouseX = 0, mouseY = 0;

        this.container.addEventListener('mousedown', (e) => {
            isMouseDown = true;
            mouseX = e.clientX;
            mouseY = e.clientY;
        });

        this.container.addEventListener('mousemove', (e) => {
            this.mouse.x = (e.clientX / this.container.clientWidth) * 2 - 1;
            this.mouse.y = -(e.clientY / this.container.clientHeight) * 2 + 1;

            if (isMouseDown) {
                const deltaX = e.clientX - mouseX;
                const deltaY = e.clientY - mouseY;
                this.rotationY += deltaX * 0.005;
                this.rotationX += deltaY * 0.005;
                this.rotationX = Math.max(-Math.PI / 2.5, Math.min(Math.PI / 2.5, this.rotationX));
                mouseX = e.clientX;
                mouseY = e.clientY;
            }
        });

        this.container.addEventListener('mouseup', () => {
            isMouseDown = false;
        });

        this.container.addEventListener('wheel', (e) => {
            e.preventDefault();
            this.zoom += e.deltaY * 0.001;
            this.zoom = Math.max(0.4, Math.min(3.0, this.zoom));
        }, { passive: false });

        this.container.addEventListener('click', (e) => {
            this.handleClick(e);
        });

        this.updateCameraPosition = () => {
            const radius = 40 * this.zoom;
            this.camera.position.x = radius * Math.sin(this.rotationY) * Math.cos(this.rotationX);
            this.camera.position.y = radius * Math.sin(this.rotationX) + 10;
            this.camera.position.z = radius * Math.cos(this.rotationY) * Math.cos(this.rotationX);
            this.camera.lookAt(this.target);
        };
    }

    setupEventListeners() {
        document.getElementById('toggle-heatmap')?.addEventListener('change', (e) => {
            this.showHeatmap = e.target.checked;
            if (this.heatmapOverlay) {
                this.heatmapOverlay.visible = this.showHeatmap;
            }
        });

        document.getElementById('toggle-cracks')?.addEventListener('change', (e) => {
            this.showCracks = e.target.checked;
            this.crackLines.forEach(line => line.visible = this.showCracks);
        });

        document.getElementById('toggle-sensors')?.addEventListener('change', (e) => {
            this.showSensors = e.target.checked;
            this.sensorMeshes.forEach(mesh => mesh.visible = this.showSensors);
        });
    }

    handleClick(event) {
        if (!this.showSensors) return;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.sensorMeshes);

        if (intersects.length > 0) {
            const clickedMesh = intersects[0].object;
            if (clickedMesh.userData.sensor) {
                if (this.onSensorClick) {
                    this.onSensorClick(clickedMesh.userData.sensor);
                }
            }
        }
    }

    setSite(site) {
        this.currentSite = site;
        document.getElementById('view-title').textContent = `${site.name} - 三维模型`;
        this.loadModelProgressively(site);
    }

    loadModelProgressively(site) {
        this.clearGrottoMeshes();
        this.isLoading = true;
        this.showLoading();

        this.updateLoadingProgress('加载骨架模型...', 20);
        setTimeout(() => {
            this.createGrottoLOD(site, 0);
            this.updateLoadingProgress('加载低精度模型...', 50);

            setTimeout(() => {
                this.createGrottoLOD(site, 1);
                this.updateLoadingProgress('加载高精度模型...', 80);

                setTimeout(() => {
                    this.createGrottoLOD(site, 2);
                    this.updateLoadingProgress('加载纹理和细节...', 95);

                    setTimeout(() => {
                        this.applyCompressedTexture(site);
                        this.isLoading = false;
                        this.hideLoading();
                    }, 150);
                }, 150);
            }, 150);
        }, 100);
    }

    updateLoadingProgress(text, percent) {
        const overlay = document.getElementById('loading-overlay');
        if (!overlay) return;
        const textEl = overlay.querySelector('.loading-text');
        const barEl = overlay.querySelector('.loading-progress-fill');
        if (textEl) textEl.textContent = text;
        if (barEl) barEl.style.width = percent + '%';
    }

    showLoading() {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.style.display = 'flex';
            const textEl = overlay.querySelector('.loading-text');
            const barEl = overlay.querySelector('.loading-progress-fill');
            if (textEl) textEl.textContent = '初始化模型...';
            if (barEl) barEl.style.width = '5%';
        }
    }

    createGrottoLOD(site, level) {
        const grottoGroup = new THREE.Group();
        grottoGroup.visible = false;
        grottoGroup.userData.lodLevel = level;

        const rockColor = this.getRockColor(site?.rock_type || '砂岩');
        const rockMaterial = this.createRockMaterial(rockColor, level);

        let detailLevel;
        switch (level) {
            case 0:
                detailLevel = 0;
                break;
            case 1:
                detailLevel = 1;
                break;
            case 2:
            default:
                detailLevel = this.isHighEndDevice() ? 2 : 1;
        }

        const rockGeometry = new THREE.DodecahedronGeometry(15, detailLevel);
        const positions = rockGeometry.attributes.position;
        for (let i = 0; i < positions.count; i++) {
            const x = positions.getX(i);
            const y = positions.getY(i);
            const z = positions.getZ(i);

            const deformAmount = level === 0 ? 1.5 : level === 1 ? 2.5 : 3.5;
            positions.setX(i, x + (Math.random() - 0.5) * deformAmount);
            positions.setY(i, Math.max(0, y + (Math.random() - 0.5) * deformAmount));
            positions.setZ(i, z + (Math.random() - 0.5) * deformAmount);
        }
        rockGeometry.computeVertexNormals();

        const mainRock = new THREE.Mesh(rockGeometry, rockMaterial);
        mainRock.castShadow = level >= 1 && this.isHighEndDevice();
        mainRock.receiveShadow = level >= 1;
        grottoGroup.add(mainRock);

        if (level >= 1) {
            this.createCaves(grottoGroup, site, level);
        }

        if (level >= 0) {
            this.createPlatform(grottoGroup, level);
        }

        if (level >= 1) {
            this.createStairs(grottoGroup, level);
        }

        this.grottoLODs.push(grottoGroup);
        this.scene.add(grottoGroup);
        this.updateActiveLOD();
    }

    createRockMaterial(color, level) {
        const materialOptions = {
            color: color,
            roughness: level === 0 ? 1.0 : level === 1 ? 0.92 : 0.88,
            metalness: level === 0 ? 0.0 : level === 1 ? 0.08 : 0.12,
        };

        if (level === 0) {
            materialOptions.flatShading = true;
            return new THREE.MeshLambertMaterial(materialOptions);
        }

        if (level === 1) {
            materialOptions.flatShading = true;
            return new THREE.MeshStandardMaterial(materialOptions);
        }

        return new THREE.MeshStandardMaterial(materialOptions);
    }

    createCompressedRockTexture(rockColor, size = 256) {
        const cacheKey = `${rockColor}_${size}`;
        if (this.textureCache.has(cacheKey)) {
            return this.textureCache.get(cacheKey);
        }

        const canvas = document.createElement('canvas');
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d');

        const r = (rockColor >> 16) & 255;
        const g = (rockColor >> 8) & 255;
        const b = rockColor & 255;

        const imageData = ctx.createImageData(size, size);
        const data = imageData.data;

        for (let i = 0; i < data.length; i += 4) {
            const x = (i / 4) % size;
            const y = Math.floor((i / 4) / size);
            const noise = (Math.sin(x * 0.1) * Math.cos(y * 0.15) + Math.random() * 0.3) * 20;

            data[i] = Math.max(0, Math.min(255, r + noise));
            data[i + 1] = Math.max(0, Math.min(255, g + noise * 0.9));
            data[i + 2] = Math.max(0, Math.min(255, b + noise * 0.8));
            data[i + 3] = 255;
        }

        ctx.putImageData(imageData, 0, 0);

        const texture = new THREE.CanvasTexture(canvas);
        texture.colorSpace = THREE.SRGBColorSpace;
        texture.wrapS = THREE.RepeatWrapping;
        texture.wrapT = THREE.RepeatWrapping;
        texture.repeat.set(2, 2);
        texture.anisotropy = Math.min(this.renderer.capabilities.getMaxAnisotropy(), 4);
        texture.needsUpdate = true;

        this.textureCache.set(cacheKey, texture);
        return texture;
    }

    applyCompressedTexture(site) {
        const rockColor = this.getRockColor(site?.rock_type || '砂岩');
        const texture = this.createCompressedRockTexture(rockColor, this.isHighEndDevice() ? 256 : 128);

        this.grottoLODs.forEach((group, level) => {
            if (level >= 1) {
                group.traverse((child) => {
                    if (child.isMesh && child.material && child.material.color) {
                        if (level >= 2) {
                            child.material = new THREE.MeshStandardMaterial({
                                map: texture,
                                color: child.material.color,
                                roughness: 0.85,
                                metalness: 0.1,
                            });
                        }
                        child.material.needsUpdate = true;
                    }
                });
            }
        });
    }

    getRockColor(rockType) {
        const colors = {
            '砂岩': 0xd4a574,
            '石灰岩': 0xb8b8b8,
            '花岗岩': 0x8b7355,
            '砂砾岩': 0xc19a6b,
        };
        return colors[rockType] || 0xd4a574;
    }

    createCaves(group, site, level) {
        const caveCount = level === 1 ? 4 : 5 + Math.floor(Math.random() * 4);
        const segments = level === 1 ? 4 : 8;

        for (let i = 0; i < caveCount; i++) {
            const angle = (i / caveCount) * Math.PI * 2;
            const radius = 12;
            const x = Math.cos(angle) * radius;
            const y = 3 + (i / caveCount) * 8;
            const z = Math.sin(angle) * radius;

            const caveGeometry = new THREE.SphereGeometry(2 + Math.random() * 1.5, segments, Math.max(3, segments - 2));
            const caveMaterial = new THREE.MeshStandardMaterial({
                color: 0x1a1a1a,
                roughness: 1,
                metalness: 0,
            });
            const cave = new THREE.Mesh(caveGeometry, caveMaterial);
            cave.position.set(x, y, z);
            group.add(cave);

            if (level >= 2) {
                const frameGeometry = new THREE.TorusGeometry(2.5, 0.3, Math.max(4, segments - 4), Math.max(8, segments * 2));
                const frameMaterial = new THREE.MeshStandardMaterial({
                    color: 0x8b7355,
                    roughness: 0.8,
                });
                const frame = new THREE.Mesh(frameGeometry, frameMaterial);
                frame.position.set(x, y, z);
                frame.lookAt(0, y, 0);
                group.add(frame);
            }
        }
    }

    createPlatform(group, level) {
        const segments = level === 0 ? 8 : level === 1 ? 16 : 32;
        const platformGeometry = new THREE.CylinderGeometry(20, 22, 1, segments);
        const platformMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b5344,
            roughness: 0.92,
        });
        const platform = new THREE.Mesh(platformGeometry, platformMaterial);
        platform.position.y = -1;
        platform.receiveShadow = level >= 1;
        group.add(platform);
    }

    createStairs(group, level) {
        const stepCount = level === 1 ? 5 : 8;
        for (let i = 0; i < stepCount; i++) {
            const stepGeometry = new THREE.BoxGeometry(6, 0.5, 2);
            const stepMaterial = new THREE.MeshStandardMaterial({
                color: 0x7a6b5a,
                roughness: 0.9,
            });
            const step = new THREE.Mesh(stepGeometry, stepMaterial);
            step.position.set(0, -0.5 + i * 0.5, 15 - i * 1.5);
            step.receiveShadow = level >= 2;
            group.add(step);
        }
    }

    updateActiveLOD() {
        const cameraDistance = this.camera.position.distanceTo(this.target);

        let targetLevel = 2;
        if (cameraDistance > this.lowPolyDistance) {
            targetLevel = 0;
        } else if (cameraDistance > this.mediumPolyDistance) {
            targetLevel = 1;
        }

        if (!this.isHighEndDevice()) {
            targetLevel = Math.min(targetLevel, 1);
        }

        if (targetLevel !== this.currentLODLevel && this.grottoLODs.length > targetLevel) {
            this.grottoLODs.forEach((g, idx) => {
                g.visible = idx === targetLevel;
            });
            this.currentLODLevel = targetLevel;
        }
    }

    setSensors(sensors) {
        this.sensors = sensors;
        this.clearSensorMeshes();
        this.clearCrackLines();
        sensors.forEach(sensor => this.createSensorMesh(sensor));
    }

    createCompressedLabelTexture(sensorCode, size = 128) {
        const cacheKey = `label_${sensorCode}_${size}`;
        if (this.textureCache.has(cacheKey)) {
            return this.textureCache.get(cacheKey);
        }

        const canvas = document.createElement('canvas');
        canvas.width = size;
        canvas.height = Math.floor(size / 4);
        const ctx = canvas.getContext('2d');

        ctx.fillStyle = 'rgba(0, 0, 0, 0.75)';
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        ctx.fillStyle = '#ffffff';
        ctx.font = `bold ${Math.floor(size / 9)}px sans-serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(sensorCode, canvas.width / 2, canvas.height / 2);

        const texture = new THREE.CanvasTexture(canvas);
        texture.colorSpace = THREE.SRGBColorSpace;
        texture.minFilter = THREE.LinearFilter;
        texture.magFilter = THREE.LinearFilter;

        this.textureCache.set(cacheKey, texture);
        return texture;
    }

    createSensorMesh(sensor) {
        const position = new THREE.Vector3(
            (sensor.position_x - 20) * 0.8,
            sensor.position_y * 0.8,
            (sensor.position_z - 20) * 0.8
        );

        let color;
        switch (sensor.sensor_type) {
            case '温度':
            case '湿度':
                color = 0x3b82f6;
                break;
            case '表面硬度':
                color = 0x22c55e;
                break;
            case '裂隙宽度':
                color = 0xef4444;
                this.createCrackLine(position);
                break;
            default:
                color = 0xf59e0b;
        }

        const segments = this.isHighEndDevice() ? 16 : 8;
        const geometry = new THREE.SphereGeometry(0.5, segments, segments);
        const material = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.9,
        });
        const mesh = new THREE.Mesh(geometry, material);
        mesh.position.copy(position);
        mesh.userData.sensor = sensor;

        if (this.isHighEndDevice()) {
            const glowGeometry = new THREE.SphereGeometry(0.8, segments, segments);
            const glowMaterial = new THREE.MeshBasicMaterial({
                color: color,
                transparent: true,
                opacity: 0.25,
            });
            const glow = new THREE.Mesh(glowGeometry, glowMaterial);
            mesh.add(glow);
        }

        const labelTexture = this.createCompressedLabelTexture(sensor.sensor_code, this.isHighEndDevice() ? 128 : 64);
        const labelMaterial = new THREE.SpriteMaterial({
            map: labelTexture,
            depthTest: false,
        });
        const label = new THREE.Sprite(labelMaterial);
        label.position.y = 1.2;
        const labelScale = this.isHighEndDevice() ? 1.0 : 0.8;
        label.scale.set(3 * labelScale, 0.75 * labelScale, 1);
        mesh.add(label);

        mesh.visible = this.showSensors;
        this.sensorMeshes.push(mesh);
        this.scene.add(mesh);

        this.animateSensor(mesh);
    }

    createCrackLine(startPosition) {
        const points = [];
        let currentPos = startPosition.clone();
        points.push(currentPos.clone());

        const segmentCount = this.isHighEndDevice() ? 6 : 4;
        for (let i = 0; i < segmentCount; i++) {
            currentPos.x += (Math.random() - 0.5) * 2;
            currentPos.y += (Math.random() - 0.5) * 1.5;
            currentPos.z += (Math.random() - 0.5) * 2;
            points.push(currentPos.clone());
        }

        const geometry = new THREE.BufferGeometry().setFromPoints(points);
        const material = new THREE.LineBasicMaterial({
            color: 0xff0000,
            transparent: true,
            opacity: 0.75,
        });
        const line = new THREE.Line(geometry, material);
        line.visible = this.showCracks;
        this.crackLines.push(line);
        this.scene.add(line);
    }

    animateSensor(mesh) {
        const animate = () => {
            if (mesh && mesh.parent) {
                const time = Date.now() * 0.002;
                mesh.position.y += Math.sin(time + mesh.position.x) * 0.002;
                mesh.rotation.y += 0.01;
            }
            this.sensorAnimationId = requestAnimationFrame(animate);
        };
        animate();
    }

    clearGrottoMeshes() {
        this.grottoLODs.forEach(group => {
            this.scene.remove(group);
            group.traverse((child) => {
                if (child.geometry) child.geometry.dispose();
                if (child.material) {
                    if (Array.isArray(child.material)) {
                        child.material.forEach(m => {
                            if (m.map) m.map.dispose();
                            m.dispose();
                        });
                    } else {
                        if (child.material.map) child.material.map.dispose();
                        child.material.dispose();
                    }
                }
            });
        });
        this.grottoLODs = [];
        this.currentLODLevel = 0;
    }

    clearSensorMeshes() {
        this.sensorMeshes.forEach(mesh => {
            this.scene.remove(mesh);
            if (mesh.geometry) mesh.geometry.dispose();
            if (mesh.material) {
                if (Array.isArray(mesh.material)) {
                    mesh.material.forEach(m => m.dispose());
                } else {
                    mesh.material.dispose();
                }
            }
        });
        this.sensorMeshes = [];
    }

    clearCrackLines() {
        this.crackLines.forEach(line => {
            this.scene.remove(line);
            if (line.geometry) line.geometry.dispose();
            if (line.material) line.material.dispose();
        });
        this.crackLines = [];
    }

    updateHeatmap(data) {
        const heatmapSize = this.isHighEndDevice() ? 512 : 256;

        if (!this.heatmapOverlay) {
            const canvas = document.createElement('canvas');
            canvas.width = heatmapSize;
            canvas.height = heatmapSize;
            const texture = new THREE.CanvasTexture(canvas);
            texture.colorSpace = THREE.SRGBColorSpace;
            texture.minFilter = THREE.LinearFilter;
            texture.magFilter = THREE.LinearFilter;

            const geometry = new THREE.PlaneGeometry(35, 35);
            const material = new THREE.MeshBasicMaterial({
                map: texture,
                transparent: true,
                opacity: 0.55,
                side: THREE.DoubleSide,
                depthWrite: false,
            });
            this.heatmapOverlay = new THREE.Mesh(geometry, material);
            this.heatmapOverlay.rotation.x = -Math.PI / 2;
            this.heatmapOverlay.position.y = 0.1;
            this.heatmapOverlay.visible = this.showHeatmap;
            this.scene.add(this.heatmapOverlay);
        }

        const canvas = this.heatmapOverlay.material.map.image;
        window.heatmapRenderer?.render(canvas, data);
        this.heatmapOverlay.material.map.needsUpdate = true;
    }

    hideLoading() {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.style.display = 'none';
        }
    }

    onResize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }

    animate() {
        this.animationId = requestAnimationFrame(() => this.animate());

        if (this.updateCameraPosition) {
            this.updateCameraPosition();
        }

        this.frameCount++;
        if (this.frameCount % this.lodCheckInterval === 0 && this.grottoLODs.length > 0) {
            this.updateActiveLOD();
        }

        this.renderer.render(this.scene, this.camera);
    }

    dispose() {
        if (this.animationId) cancelAnimationFrame(this.animationId);
        if (this.sensorAnimationId) cancelAnimationFrame(this.sensorAnimationId);

        this.clearGrottoMeshes();
        this.clearSensorMeshes();
        this.clearCrackLines();

        this.textureCache.forEach((tex) => tex.dispose());
        this.textureCache.clear();

        if (this.defaultTexture) this.defaultTexture.dispose();

        if (this.renderer) {
            this.renderer.dispose();
            if (this.renderer.domElement.parentNode) {
                this.renderer.domElement.parentNode.removeChild(this.renderer.domElement);
            }
        }
    }
}
