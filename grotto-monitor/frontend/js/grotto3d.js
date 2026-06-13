class Grotto3DViewer {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.sensors = [];
        this.sensorMeshes = [];
        this.crackLines = [];
        this.grottoMesh = null;
        this.heatmapOverlay = null;
        this.showHeatmap = true;
        this.showCracks = true;
        this.showSensors = true;
        this.currentSite = null;
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.onSensorClick = null;
        
        this.init();
    }

    init() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0a0a0f);
        this.scene.fog = new THREE.Fog(0x0a0a0f, 50, 150);

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.set(30, 20, 30);
        this.camera.lookAt(0, 0, 0);

        this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(window.devicePixelRatio);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.container.appendChild(this.renderer.domElement);

        this.setupLighting();
        this.setupControls();
        this.setupEventListeners();
        this.animate();

        window.addEventListener('resize', () => this.onResize());
    }

    setupLighting() {
        const ambientLight = new THREE.AmbientLight(0x404050, 0.5);
        this.scene.add(ambientLight);

        const mainLight = new THREE.DirectionalLight(0xffffff, 1.2);
        mainLight.position.set(50, 50, 30);
        mainLight.castShadow = true;
        mainLight.shadow.mapSize.width = 2048;
        mainLight.shadow.mapSize.height = 2048;
        mainLight.shadow.camera.near = 0.5;
        mainLight.shadow.camera.far = 200;
        mainLight.shadow.camera.left = -50;
        mainLight.shadow.camera.right = 50;
        mainLight.shadow.camera.top = 50;
        mainLight.shadow.camera.bottom = -50;
        this.scene.add(mainLight);

        const fillLight = new THREE.DirectionalLight(0x4a90d9, 0.3);
        fillLight.position.set(-30, 20, -30);
        this.scene.add(fillLight);

        const rimLight = new THREE.DirectionalLight(0xff9500, 0.4);
        rimLight.position.set(-20, 10, 40);
        this.scene.add(rimLight);
    }

    setupControls() {
        let isMouseDown = false;
        let mouseX = 0, mouseY = 0;
        let targetX = 0, targetY = 0;
        let rotationX = 0, rotationY = 0;
        let zoom = 1;
        const target = new THREE.Vector3(0, 0, 0);

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
                rotationY += deltaX * 0.005;
                rotationX += deltaY * 0.005;
                rotationX = Math.max(-Math.PI / 2.5, Math.min(Math.PI / 2.5, rotationX));
                mouseX = e.clientX;
                mouseY = e.clientY;
            }
        });

        this.container.addEventListener('mouseup', () => {
            isMouseDown = false;
        });

        this.container.addEventListener('wheel', (e) => {
            e.preventDefault();
            zoom += e.deltaY * 0.001;
            zoom = Math.max(0.5, Math.min(2.5, zoom));
        });

        this.container.addEventListener('click', (e) => {
            this.handleClick(e);
        });

        this.updateCameraPosition = () => {
            const radius = 40 * zoom;
            this.camera.position.x = radius * Math.sin(rotationY) * Math.cos(rotationX);
            this.camera.position.y = radius * Math.sin(rotationX) + 10;
            this.camera.position.z = radius * Math.cos(rotationY) * Math.cos(rotationX);
            this.camera.lookAt(target);
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

    createGrottoModel(site) {
        if (this.grottoMesh) {
            this.scene.remove(this.grottoMesh);
        }

        const grottoGroup = new THREE.Group();

        const rockGeometry = new THREE.DodecahedronGeometry(15, 1);
        const positions = rockGeometry.attributes.position;
        for (let i = 0; i < positions.count; i++) {
            const x = positions.getX(i);
            const y = positions.getY(i);
            const z = positions.getZ(i);
            
            positions.setX(i, x + (Math.random() - 0.5) * 3);
            positions.setY(i, Math.max(0, y + (Math.random() - 0.5) * 3));
            positions.setZ(i, z + (Math.random() - 0.5) * 3);
        }
        rockGeometry.computeVertexNormals();

        const rockColor = this.getRockColor(site?.rock_type || '砂岩');
        const rockMaterial = new THREE.MeshStandardMaterial({
            color: rockColor,
            roughness: 0.9,
            metalness: 0.1,
            flatShading: true,
        });

        const mainRock = new THREE.Mesh(rockGeometry, rockMaterial);
        mainRock.castShadow = true;
        mainRock.receiveShadow = true;
        grottoGroup.add(mainRock);

        this.createCaves(grottoGroup, site);
        this.createPlatform(grottoGroup);
        this.createStairs(grottoGroup);

        this.grottoMesh = grottoGroup;
        this.scene.add(grottoGroup);

        this.hideLoading();
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

    createCaves(group, site) {
        const caveCount = 5 + Math.floor(Math.random() * 4);
        for (let i = 0; i < caveCount; i++) {
            const angle = (i / caveCount) * Math.PI * 2;
            const radius = 12;
            const x = Math.cos(angle) * radius;
            const y = 3 + Math.random() * 8;
            const z = Math.sin(angle) * radius;

            const caveGeometry = new THREE.SphereGeometry(2 + Math.random() * 1.5, 8, 6);
            const caveMaterial = new THREE.MeshStandardMaterial({
                color: 0x1a1a1a,
                roughness: 1,
                metalness: 0,
            });
            const cave = new THREE.Mesh(caveGeometry, caveMaterial);
            cave.position.set(x, y, z);
            group.add(cave);

            const frameGeometry = new THREE.TorusGeometry(2.5, 0.3, 8, 16);
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

    createPlatform(group) {
        const platformGeometry = new THREE.CylinderGeometry(20, 22, 1, 32);
        const platformMaterial = new THREE.MeshStandardMaterial({
            color: 0x6b5344,
            roughness: 0.9,
        });
        const platform = new THREE.Mesh(platformGeometry, platformMaterial);
        platform.position.y = -1;
        platform.receiveShadow = true;
        group.add(platform);
    }

    createStairs(group) {
        for (let i = 0; i < 8; i++) {
            const stepGeometry = new THREE.BoxGeometry(6, 0.5, 2);
            const stepMaterial = new THREE.MeshStandardMaterial({
                color: 0x7a6b5a,
                roughness: 0.9,
            });
            const step = new THREE.Mesh(stepGeometry, stepMaterial);
            step.position.set(0, -0.5 + i * 0.5, 15 - i * 1.5);
            step.receiveShadow = true;
            group.add(step);
        }
    }

    setSensors(sensors) {
        this.sensors = sensors;
        this.clearSensorMeshes();
        this.clearCrackLines();
        sensors.forEach(sensor => this.createSensorMesh(sensor));
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

        const geometry = new THREE.SphereGeometry(0.5, 16, 16);
        const material = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.9,
        });
        const mesh = new THREE.Mesh(geometry, material);
        mesh.position.copy(position);
        mesh.userData.sensor = sensor;

        const glowGeometry = new THREE.SphereGeometry(0.8, 16, 16);
        const glowMaterial = new THREE.MeshBasicMaterial({
            color: color,
            transparent: true,
            opacity: 0.3,
        });
        const glow = new THREE.Mesh(glowGeometry, glowMaterial);
        mesh.add(glow);

        const labelCanvas = document.createElement('canvas');
        labelCanvas.width = 128;
        labelCanvas.height = 32;
        const ctx = labelCanvas.getContext('2d');
        ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        ctx.fillRect(0, 0, 128, 32);
        ctx.fillStyle = '#ffffff';
        ctx.font = 'bold 14px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText(sensor.sensor_code, 64, 22);
        
        const labelTexture = new THREE.CanvasTexture(labelCanvas);
        const labelMaterial = new THREE.SpriteMaterial({ map: labelTexture });
        const label = new THREE.Sprite(labelMaterial);
        label.position.y = 1.2;
        label.scale.set(3, 0.75, 1);
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

        for (let i = 0; i < 5; i++) {
            currentPos.x += (Math.random() - 0.5) * 2;
            currentPos.y += (Math.random() - 0.5) * 1.5;
            currentPos.z += (Math.random() - 0.5) * 2;
            points.push(currentPos.clone());
        }

        const geometry = new THREE.BufferGeometry().setFromPoints(points);
        const material = new THREE.LineBasicMaterial({
            color: 0xff0000,
            linewidth: 2,
            transparent: true,
            opacity: 0.8,
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
            requestAnimationFrame(animate);
        };
        animate();
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
        if (!this.heatmapOverlay) {
            const canvas = document.createElement('canvas');
            canvas.width = 512;
            canvas.height = 512;
            const texture = new THREE.CanvasTexture(canvas);
            
            const geometry = new THREE.PlaneGeometry(35, 35);
            const material = new THREE.MeshBasicMaterial({
                map: texture,
                transparent: true,
                opacity: 0.6,
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
        const loading = document.getElementById('loading-overlay');
        if (loading) {
            loading.style.display = 'none';
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
        requestAnimationFrame(() => this.animate());
        
        if (this.updateCameraPosition) {
            this.updateCameraPosition();
        }
        
        this.renderer.render(this.scene, this.camera);
    }

    setSite(site) {
        this.currentSite = site;
        this.createGrottoModel(site);
        document.getElementById('view-title').textContent = `${site.name} - 三维模型`;
    }
}
