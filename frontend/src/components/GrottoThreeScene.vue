<template>
  <div class="grotto-3d-container" ref="containerRef">
    <canvas ref="canvasRef" class="three-canvas" />

    <div class="loading-overlay" v-if="loadingState.phase !== 'complete' && loadingState.phase !== 'idle'">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">{{ loadingState.phase === 'skeleton' ? '加载骨架...' : '加载细节...' }}</div>
        <div class="loading-bar">
          <div class="loading-bar-fill" :style="{ width: loadingState.progress + '%' }"></div>
        </div>
        <div class="loading-detail">LOD 级别: {{ loadingState.detailLevel }} / 3</div>
      </div>
    </div>

    <div class="controls-panel">
      <div class="control-group">
        <div class="control-label">显示选项</div>
        <el-checkbox v-model="showPoints" size="small">监测点</el-checkbox>
        <el-checkbox v-model="showHeatmap" size="small">风化热力图</el-checkbox>
        <el-checkbox v-model="showCracks" size="small">裂隙标记</el-checkbox>
        <el-checkbox v-model="showLabels" size="small">标签</el-checkbox>
      </div>
      <div class="control-group">
        <div class="control-label">视角</div>
        <el-radio-group v-model="viewPreset" size="small" @change="applyViewPreset">
          <el-radio-button value="front">正视</el-radio-button>
          <el-radio-button value="top">俯视</el-radio-button>
          <el-radio-button value="side">侧视</el-radio-button>
          <el-radio-button value="iso">立体</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <div class="info-panel" v-if="cave">
      <div class="info-title">
        <el-icon><Location /></el-icon>
        {{ cave.name }}
      </div>
      <div class="info-row"><span>岩石类型</span><b>{{ cave.rockType }}</b></div>
      <div class="info-row"><span>朝代</span><b>{{ cave.dynasty }}</b></div>
      <div class="info-row"><span>监测点</span><b class="accent">{{ points.length }}个</b></div>
      <div class="info-row"><span>活跃告警</span><b class="danger">{{ activeAlertsCount }}</b></div>
    </div>

    <div class="stats-legend">
      <div class="legend-item">
        <span class="dot normal"></span><span>正常</span>
      </div>
      <div class="legend-item">
        <span class="dot warning"></span><span>关注</span>
      </div>
      <div class="legend-item">
        <span class="dot danger"></span><span>告警</span>
      </div>
    </div>
  </div>
</template>

<script setup>import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick, shallowRef } from 'vue';
import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import { Location } from '@element-plus/icons-vue';
const props = defineProps({
 cave: Object,
 points: { type: Array, default: () => [] },
 sensorData: { type: Array, default: () => [] }
});
const emit = defineEmits(['point-click']);
const containerRef = ref(null);
const canvasRef = ref(null);
const showPoints = ref(true);
const showHeatmap = ref(true);
const showCracks = ref(true);
const showLabels = ref(true);
const viewPreset = ref('iso');
const scene = shallowRef(null);
const camera = shallowRef(null);
const renderer = shallowRef(null);
const controls = shallowRef(null);
let animationId = null;
const loadingState = reactive({
 phase: 'idle',
 progress: 0,
 detailLevel: 0
});
const objects = reactive({
 caveGroup: null,
 pointsGroup: null,
 heatmapGroup: null,
 cracksGroup: null,
 labels: []
});
const raycaster = new THREE.Raycaster();
const mouse = new THREE.Vector2();
const activeAlertsCount = computed(() => {
 return props.points.filter(pt => {
 const sensor = props.sensorData.find(s => s.pointId === pt.id);
 if (!sensor) return false;
 const drop = pt.initialHardness > 0 ? (pt.initialHardness - sensor.surfaceHardness) / pt.initialHardness * 100 : 0;
 return sensor.crackWidth > 0.5 || drop > 20;
 }).length;
});
function initThree() {
 if (!containerRef.value || !canvasRef.value) return;
 const rect = containerRef.value.getBoundingClientRect();
 const width = rect.width || 800;
 const height = rect.height || 600;
 scene.value = new THREE.Scene();
 scene.value.background = new THREE.Color(0x0a0f1a);
 scene.value.fog = new THREE.FogExp2(0x0a0f1a, 0.004);
 camera.value = new THREE.PerspectiveCamera(55, width / height, 0.1, 2000);
 camera.value.position.set(80, 60, 80);
 renderer.value = new THREE.WebGLRenderer({
 canvas: canvasRef.value,
 antialias: true,
 alpha: true,
 powerPreference: 'high-performance'
 });
 renderer.value.setPixelRatio(Math.min(devicePixelRatio, 2));
 renderer.value.setSize(width, height);
 renderer.value.shadowMap.enabled = true;
 renderer.value.shadowMap.type = THREE.PCFSoftShadowMap;
 renderer.value.toneMapping = THREE.ACESFilmicToneMapping;
 renderer.value.toneMappingExposure = 1.1;
 controls.value = new OrbitControls(camera.value, renderer.value.domElement);
 controls.value.enableDamping = true;
 controls.value.dampingFactor = 0.08;
 controls.value.minDistance = 30;
 controls.value.maxDistance = 300;
 controls.value.maxPolarAngle = Math.PI / 2 + 0.1;
 controls.value.target.set(0, 10, 0);
 addLights();
 createGround();
 animate();
 progressiveLoad();
}
async function progressiveLoad() {
 loadingState.phase = 'skeleton';
 loadingState.progress = 0;
 createCaveSkeleton();
 await yieldFrame();
 loadingState.phase = 'detail';
 loadingState.progress = 30;
 await loadDetailLevel1();
 loadingState.progress = 60;
 await loadDetailLevel2();
 loadingState.progress = 90;
 await loadDetailLevel3();
 loadingState.progress = 100;
 loadingState.phase = 'complete';
 loadingState.detailLevel = 3;
 createOrUpdatePoints();
 createOrUpdateHeatmap();
 createOrUpdateCracks();
}
function yieldFrame() {
 return new Promise(resolve => requestAnimationFrame(resolve));
}
function createCaveSkeleton() {
 objects.caveGroup = new THREE.Group();
 const skeletonMat = new THREE.MeshBasicMaterial({
 color: 0x5a4a3a,
 wireframe: true,
 transparent: true,
 opacity: 0.3
 });
 const mainGeo = new THREE.SphereGeometry(55, 6, 4);
 const mainHill = new THREE.Mesh(mainGeo, skeletonMat);
 mainHill.position.y = 10;
 objects.caveGroup.add(mainHill);
 for (let i = 0; i < 7; i++) {
  const angle = (i / 7) * Math.PI * 2 + 0.3;
  const dist = 45 + 12;
  const hillGeo = new THREE.SphereGeometry(18, 4, 3);
  const hill = new THREE.Mesh(hillGeo, skeletonMat);
  hill.position.set(Math.cos(angle) * dist, 8, Math.sin(angle) * dist);
  objects.caveGroup.add(hill);
 }
 scene.value.add(objects.caveGroup);
}
async function loadDetailLevel1() {
 if (!objects.caveGroup) return;
 clearCaveGroup();
 const rockMat = new THREE.MeshStandardMaterial({
 color: 0x8b7355,
 roughness: 0.9,
 metalness: 0.05,
 flatShading: true
 });
 const mainHillGeo = new THREE.SphereGeometry(55, 10, 8);
 const positions = mainHillGeo.attributes.position;
 for (let i = 0; i < positions.count; i++) {
  const x = positions.getX(i), y = positions.getY(i), z = positions.getZ(i);
  const noise1 = Math.sin(x * 0.08) * Math.cos(z * 0.09) * 8;
  const noise2 = Math.sin(y * 0.1 + x * 0.05) * 5;
  const flatten = y < 0 ? -y * 0.9 : 0;
  positions.setXYZ(i, x + noise1, y * 0.72 + noise2 + flatten, z + noise1 * 0.7);
 }
 positions.needsUpdate = true;
 mainHillGeo.computeVertexNormals();
 const mainHill = new THREE.Mesh(mainHillGeo, rockMat);
 mainHill.position.y = 10;
 mainHill.castShadow = true;
 mainHill.receiveShadow = true;
 mainHill.userData.lodLevel = 1;
 objects.caveGroup.add(mainHill);
 for (let i = 0; i < 7; i++) {
  const angle = (i / 7) * Math.PI * 2 + 0.3;
  const dist = 45 + Math.random() * 15;
  const hillGeo = new THREE.SphereGeometry(18 + Math.random() * 12, 8, 6);
  const pos2 = hillGeo.attributes.position;
  for (let j = 0; j < pos2.count; j++) {
   const x = pos2.getX(j), y = pos2.getY(j), z = pos2.getZ(j);
   const n = Math.sin(x * 0.15) * Math.cos(z * 0.12) * 3;
   const flat = y < 0 ? -y * 0.95 : 0;
   pos2.setXYZ(j, x + n, y * 0.55 + n * 0.3 + flat, z + n * 0.8);
  }
  pos2.needsUpdate = true;
  hillGeo.computeVertexNormals();
  const hill = new THREE.Mesh(hillGeo, rockMat);
  hill.position.set(Math.cos(angle) * dist, 6 + Math.random() * 4, Math.sin(angle) * dist);
  hill.castShadow = true;
  hill.receiveShadow = true;
  hill.userData.lodLevel = 1;
  objects.caveGroup.add(hill);
 }
 scene.value.add(objects.caveGroup);
 await yieldFrame();
}
async function loadDetailLevel2() {
 if (!objects.caveGroup) return;
 const rockMatDark = new THREE.MeshStandardMaterial({
  color: 0x6b5344,
  roughness: 0.95,
  metalness: 0.0,
  flatShading: true
 });
 const nichePositions = [];
 for (let i = 0; i < 12; i++) {
  const angle = (i / 12) * Math.PI * 2 + 0.2;
  const dist = 28 + Math.random() * 18;
  const nx = Math.cos(angle) * dist;
  const nz = Math.sin(angle) * dist;
  nichePositions.push({ angle, dist, nx, nz });
 }
 for (let batch = 0; batch < 4; batch++) {
  const start = batch * 3;
  const end = Math.min(start + 3, nichePositions.length);
  for (let i = start; i < end; i++) {
   const { nx, nz } = nichePositions[i];
   const nicheDepth = 4 + Math.random() * 4;
   const nicheWidth = 5 + Math.random() * 4;
   const nicheHeight = 7 + Math.random() * 5;
   const nicheGeo = new THREE.SphereGeometry(nicheWidth, 6, 4);
   const nichePos = nicheGeo.attributes.position;
   for (let j = 0; j < nichePos.count; j++) {
    let x = nichePos.getX(j), y = nichePos.getY(j), z = nichePos.getZ(j);
    if (z > 0) { z *= 0.2; }
    nichePos.setXYZ(j, x, y * (nicheHeight / nicheWidth), z * (nicheDepth / nicheWidth));
   }
   nichePos.needsUpdate = true;
   nicheGeo.computeVertexNormals();
   const nicheMat = new THREE.MeshStandardMaterial({
    color: 0x1a1410,
    roughness: 1.0,
    side: THREE.BackSide
   });
   const niche = new THREE.Mesh(nicheGeo, nicheMat);
   niche.position.set(nx, 8 + Math.random() * 15, nz);
   niche.lookAt(0, niche.position.y, 0);
   niche.rotateY(Math.PI);
   niche.userData.lodLevel = 2;
   objects.caveGroup.add(niche);
  }
  await yieldFrame();
 }
 for (let i = 0; i < 4; i++) {
  const pillarGeo = new THREE.CylinderGeometry(1.8, 2.5, 28 + Math.random() * 12, 6);
  const pillar = new THREE.Mesh(pillarGeo, rockMatDark);
  const angle = (i / 4) * Math.PI * 2 + 0.7;
  pillar.position.set(Math.cos(angle) * 35, 18, Math.sin(angle) * 35);
  pillar.castShadow = true;
  pillar.receiveShadow = true;
  pillar.userData.lodLevel = 2;
  objects.caveGroup.add(pillar);
 }
}
async function loadDetailLevel3() {
 if (!objects.caveGroup) return;
 const lod1Meshes = [];
 objects.caveGroup.traverse(obj => {
  if (obj.isMesh && obj.userData.lodLevel === 1) {
   lod1Meshes.push(obj);
  }
 });
 for (const mesh of lod1Meshes) {
  const oldGeo = mesh.geometry;
  const baseParams = oldGeo.parameters || {};
  const newWidthSegs = Math.min((baseParams.widthSegments || 10) * 2, 32);
  const newHeightSegs = Math.min((baseParams.heightSegments || 8) * 2, 24);
  const newGeo = new THREE.SphereGeometry(
   baseParams.radius || 55,
   newWidthSegs,
   newHeightSegs
  );
  const positions = newGeo.attributes.position;
  for (let i = 0; i < positions.count; i++) {
   const x = positions.getX(i), y = positions.getY(i), z = positions.getZ(i);
   const noise1 = Math.sin(x * 0.08) * Math.cos(z * 0.09) * 8;
   const noise2 = Math.sin(y * 0.1 + x * 0.05) * 5;
   const noise3 = Math.sin(x * 0.17 + z * 0.13) * 2.5;
   const flatten = y < 0 ? -y * 0.9 : 0;
   positions.setXYZ(i, x + noise1 + noise3 * 0.3, y * 0.72 + noise2 + flatten, z + noise1 * 0.7 + noise3 * 0.2);
  }
  positions.needsUpdate = true;
  newGeo.computeVertexNormals();
  mesh.geometry.dispose();
  mesh.geometry = newGeo;
  mesh.userData.lodLevel = 3;
  await yieldFrame();
 }
}
function clearCaveGroup() {
 if (!objects.caveGroup) return;
 objects.caveGroup.traverse(obj => {
  obj.geometry?.dispose?.();
  if (Array.isArray(obj.material)) obj.material.forEach(m => m.dispose());
  else obj.material?.dispose?.();
 });
 scene.value.remove(objects.caveGroup);
 objects.caveGroup = new THREE.Group();
}
function addLights() {
 const ambient = new THREE.AmbientLight(0x404a5c, 0.6);
 scene.value.add(ambient);
 const hemisphere = new THREE.HemisphereLight(0xffd699, 0x1a2744, 0.5);
 scene.value.add(hemisphere);
 const dir = new THREE.DirectionalLight(0xfff4d6, 1.2);
 dir.position.set(60, 100, 40);
 dir.castShadow = true;
 dir.shadow.mapSize.width = 2048;
 dir.shadow.mapSize.height = 2048;
 dir.shadow.camera.near = 0.5;
 dir.shadow.camera.far = 500;
 dir.shadow.camera.left = -120;
 dir.shadow.camera.right = 120;
 dir.shadow.camera.top = 120;
 dir.shadow.camera.bottom = -120;
 scene.value.add(dir);
 const warmLight = new THREE.PointLight(0xff9933, 0.8, 150);
 warmLight.position.set(0, 40, 0);
 scene.value.add(warmLight);
}
function createGround() {
 const groundGeo = new THREE.CircleGeometry(150, 64);
 const groundMat = new THREE.MeshStandardMaterial({
 color: 0x1a1f2e,
 roughness: 0.95,
 metalness: 0.0
 });
 const ground = new THREE.Mesh(groundGeo, groundMat);
 ground.rotation.x = -Math.PI / 2;
 ground.receiveShadow = true;
 scene.value.add(ground);
 const gridHelper = new THREE.GridHelper(240, 60, 0x334155, 0x1e293b);
 gridHelper.position.y = 0.01;
 scene.value.add(gridHelper);
}
function getPointStatus(pt) {
 const sensor = props.sensorData.find(s => s.pointId === pt.id);
 const hardness = sensor?.surfaceHardness ?? pt.initialHardness;
 const crack = sensor?.crackWidth ?? pt.initialCrackWidth;
 const drop = pt.initialHardness > 0 ? (pt.initialHardness - hardness) / pt.initialHardness * 100 : 0;
 let status = 'normal';
 let score = Math.min(100, (drop * 3 + crack * 80) / 4);
 if (crack > 0.5 || drop > 20) status = 'danger';
 else if (crack > 0.3 || drop > 12) status = 'warning';
 else if (!sensor) status = 'unknown';
 return { status, score, hardness, crack, sensor };
}
function createOrUpdatePoints() {
 if (!scene.value) return;
 if (objects.pointsGroup) {
 scene.value.remove(objects.pointsGroup);
 objects.pointsGroup.traverse(obj => {
 obj.geometry?.dispose?.();
 if (Array.isArray(obj.material)) obj.material.forEach(m => m.dispose());
 else obj.material?.dispose?.();
 });
 }
 objects.pointsGroup = new THREE.Group();
 if (!showPoints.value) {
 scene.value.add(objects.pointsGroup);
 return;
 }
 props.points.forEach(pt => {
 const { status, score, hardness, crack } = getPointStatus(pt);
 const scale = status === 'danger' ? 1.8 : status === 'warning' ? 1.4 : 1;
 const colorMap = {
 normal: 0x10b981,
 warning: 0xf59e0b,
 danger: 0xef4444,
 unknown: 0x64748b
 };
 const group = new THREE.Group();
 group.userData = { pointId: pt.id, pointName: pt.name };
 const mainGeo = new THREE.SphereGeometry(1.5 * scale, 16, 16);
 const mainMat = new THREE.MeshStandardMaterial({
 color: colorMap[status],
 emissive: colorMap[status],
 emissiveIntensity: status === 'danger' ? 0.7 : status === 'warning' ? 0.4 : 0.2,
 roughness: 0.4,
 metalness: 0.3
 });
 const mainMesh = new THREE.Mesh(mainGeo, mainMat);
 group.add(mainMesh);
 if (status === 'danger') {
 const ringGeo = new THREE.RingGeometry(2.2, 2.6, 32);
 const ringMat = new THREE.MeshBasicMaterial({
 color: 0xef4444,
 side: THREE.DoubleSide,
 transparent: true,
 opacity: 0.6
 });
 const ring = new THREE.Mesh(ringGeo, ringMat);
 ring.rotation.x = -Math.PI / 2;
 ring.userData.isPulse = true;
 group.add(ring);
 }
 const pillarGeo = new THREE.CylinderGeometry(0.12, 0.15, 6, 8);
 const pillarMat = new THREE.MeshStandardMaterial({ color: 0x94a3b8, roughness: 0.6 });
 const pillar = new THREE.Mesh(pillarGeo, pillarMat);
 pillar.position.y = -3;
 group.add(pillar);
 group.position.set(pt.positionX * 1.2, pt.positionZ * 0.5 + 3, pt.positionY * 1.2);
 objects.pointsGroup.add(group);
 });
 scene.value.add(objects.pointsGroup);
}
function createOrUpdateHeatmap() {
 if (!scene.value) return;
 if (objects.heatmapGroup) {
 scene.value.remove(objects.heatmapGroup);
 objects.heatmapGroup.traverse(obj => {
 obj.geometry?.dispose?.();
 if (Array.isArray(obj.material)) obj.material.forEach(m => m.dispose());
 else obj.material?.dispose?.();
 });
 objects.heatmapGroup = null;
 }
 if (!showHeatmap.value || props.points.length === 0) return;
 objects.heatmapGroup = new THREE.Group();
 props.points.forEach(pt => {
 const { score } = getPointStatus(pt);
 if (score < 5) return;
 const normalized = Math.min(1, score / 60);
 const canvas = document.createElement('canvas');
  const texSize = Math.max(32, Math.min(128, Math.round(64 + score * 0.6)));
  canvas.width = texSize; canvas.height = texSize;
 const ctx = canvas.getContext('2d');
 const half = texSize / 2;
 const grad = ctx.createRadialGradient(half, half, 0, half, half, half);
 grad.addColorStop(0, `rgba(239,68,68,${0.6 * normalized})`);
 grad.addColorStop(0.35, `rgba(245,158,11,${0.45 * normalized})`);
 grad.addColorStop(0.65, `rgba(59,130,246,${0.25 * normalized})`);
 grad.addColorStop(1, 'rgba(16,185,129,0)');
 ctx.fillStyle = grad;
 ctx.fillRect(0, 0, texSize, texSize);
 const texture = new THREE.CanvasTexture(canvas);
 texture.generateMipmaps = false;
 texture.minFilter = THREE.LinearFilter;
 texture.magFilter = THREE.LinearFilter;
 const planeGeo = new THREE.PlaneGeometry(20 + score * 0.4, 20 + score * 0.4);
 const planeMat = new THREE.MeshBasicMaterial({
 map: texture,
 transparent: true,
 depthWrite: false,
 side: THREE.DoubleSide
 });
 const plane = new THREE.Mesh(planeGeo, planeMat);
 plane.rotation.x = -Math.PI / 2;
 plane.position.set(pt.positionX * 1.2, 0.2, pt.positionY * 1.2);
 objects.heatmapGroup.add(plane);
 });
 scene.value.add(objects.heatmapGroup);
}
function createOrUpdateCracks() {
 if (!scene.value) return;
 if (objects.cracksGroup) {
 scene.value.remove(objects.cracksGroup);
 objects.cracksGroup.traverse(obj => {
 obj.geometry?.dispose?.();
 if (Array.isArray(obj.material)) obj.material.forEach(m => m.dispose());
 else obj.material?.dispose?.();
 });
 objects.cracksGroup = null;
 }
 if (!showCracks.value) return;
 objects.cracksGroup = new THREE.Group();
 props.points.forEach(pt => {
 const { crack } = getPointStatus(pt);
 if (crack <= 0.05) return;
 const severity = Math.min(1, crack / 0.8);
 const color = severity > 0.7 ? 0xef4444 : severity > 0.4 ? 0xf59e0b : 0x3b82f6;
 const startPos = new THREE.Vector3(pt.positionX * 1.2, pt.positionZ * 0.5 + 2, pt.positionY * 1.2);
 const crackCount = crack > 0.2 ? 3 : 1;
 for (let c = 0; c < crackCount; c++) {
 const len = 4 + crack * 40;
 const angle = ((pt.id * 137.5 + c * 70) % 360) * Math.PI / 180;
 const pitch = -0.2 - Math.random() * 0.5;
 const points = [];
 const steps = 12;
 for (let i = 0; i <= steps; i++) {
 const t = i / steps;
 const progress = len * t;
 const noiseX = (Math.sin(i * 12.9898 + pt.id + c) * 43758.5453 % 1) * 1.5;
 const noiseY = (Math.cos(i * 78.233 + pt.id + c) * 2.3) * 1.2;
 const px = startPos.x + Math.cos(angle) * Math.cos(pitch) * progress + noiseX;
 const py = startPos.y + Math.sin(pitch) * progress + noiseY;
 const pz = startPos.z + Math.sin(angle) * Math.cos(pitch) * progress + noiseX * 0.6;
 points.push(new THREE.Vector3(px, py, pz));
 }
 const curve = new THREE.CatmullRomCurve3(points);
 const tubeGeo = new THREE.TubeGeometry(curve, 30, 0.08 + severity * 0.12, 6, false);
 const tubeMat = new THREE.MeshBasicMaterial({
 color,
 transparent: true,
 opacity: 0.7 + severity * 0.3
 });
 const tube = new THREE.Mesh(tubeGeo, tubeMat);
 objects.cracksGroup.add(tube);
 const glowGeo = new THREE.TubeGeometry(curve, 20, 0.25 + severity * 0.2, 4, false);
 const glowMat = new THREE.MeshBasicMaterial({
 color,
 transparent: true,
 opacity: 0.12 + severity * 0.1
 });
 objects.cracksGroup.add(new THREE.Mesh(glowGeo, glowMat));
 }
 });
 scene.value.add(objects.cracksGroup);
}
function animate() {
 animationId = requestAnimationFrame(animate);
 if (objects.pointsGroup) {
 const t = performance.now() * 0.001;
 objects.pointsGroup.traverse(obj => {
 if (obj.userData.isPulse) {
 const s = 1 + Math.sin(t * 4) * 0.15;
 obj.scale.set(s, s, 1);
 obj.material.opacity = 0.35 + Math.sin(t * 4) * 0.25;
 }
 });
 }
 controls.value?.update();
 renderer.value.render(scene.value, camera.value);
}
function applyViewPreset() {
 if (!camera.value || !controls.value) return;
 const presets = {
 front: { pos: [0, 30, 100], target: [0, 10, 0] },
 top: { pos: [0, 120, 0.1], target: [0, 0, 0] },
 side: { pos: [100, 30, 0], target: [0, 10, 0] },
 iso: { pos: [80, 60, 80], target: [0, 10, 0] }
 };
 const p = presets[viewPreset.value] || presets.iso;
 const startPos = camera.value.position.clone();
 const startTarget = controls.value.target.clone();
 const endPos = new THREE.Vector3(...p.pos);
 const endTarget = new THREE.Vector3(...p.target);
 let progress = 0;
 const anim = () => {
 progress += 0.04;
 if (progress >= 1) {
 camera.value.position.copy(endPos);
 controls.value.target.copy(endTarget);
 return;
 }
 const t = progress * progress * (3 - 2 * progress);
 camera.value.position.lerpVectors(startPos, endPos, t);
 controls.value.target.lerpVectors(startTarget, endTarget, t);
 requestAnimationFrame(anim);
 };
 anim();
}
function handleResize() {
 if (!containerRef.value || !camera.value || !renderer.value) return;
 const rect = containerRef.value.getBoundingClientRect();
 const width = rect.width || 800;
 const height = rect.height || 600;
 camera.value.aspect = width / height;
 camera.value.updateProjectionMatrix();
 renderer.value.setSize(width, height);
}
function handleClick(event) {
 if (!containerRef.value || !camera.value || !objects.pointsGroup) return;
 const rect = containerRef.value.getBoundingClientRect();
 mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
 mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;
 raycaster.setFromCamera(mouse, camera.value);
 const meshes = [];
 objects.pointsGroup.traverse(obj => {
 if (obj.isMesh && obj.parent?.userData?.pointId) {
 meshes.push(obj);
 }
 });
 const hits = raycaster.intersectObjects(meshes, false);
 if (hits.length > 0) {
 let target = hits[0].object;
 while (target && !target.userData?.pointId) target = target.parent;
 if (target?.userData?.pointId) {
 emit('point-click', target.userData.pointId);
 }
 }
}
watch([showPoints, showHeatmap, showCracks, () => props.points, () => props.sensorData], () => {
 nextTick(() => {
 createOrUpdatePoints();
 createOrUpdateHeatmap();
 createOrUpdateCracks();
 });
}, { deep: true });
let ro = null;
onMounted(() => {
 nextTick(() => {
 initThree();
 containerRef.value.addEventListener('click', handleClick);
 ro = new ResizeObserver(handleResize);
 ro.observe(containerRef.value);
 });
});
onBeforeUnmount(() => {
 cancelAnimationFrame(animationId);
 ro?.disconnect?.();
 containerRef.value?.removeEventListener('click', handleClick);
 renderer.value?.dispose?.();
});
</script>

<style lang="scss" scoped>
.grotto-3d-container {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 10px;
  overflow: hidden;
  background: linear-gradient(180deg, #0a0f1a 0%, #020617 100%);

  .loading-overlay {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    z-index: 20;
    background: rgba(2, 6, 23, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(4px);

    .loading-content {
      text-align: center;

      .loading-spinner {
        width: 40px; height: 40px;
        border: 3px solid rgba(212,175,55,0.2);
        border-top-color: #d4af37;
        border-radius: 50%;
        margin: 0 auto 14px;
        animation: spin 0.8s linear infinite;
      }

      .loading-text {
        font-size: 14px;
        color: #e2e8f0;
        margin-bottom: 10px;
      }

      .loading-bar {
        width: 200px;
        height: 4px;
        background: rgba(255,255,255,0.1);
        border-radius: 2px;
        margin: 0 auto 8px;
        overflow: hidden;

        .loading-bar-fill {
          height: 100%;
          background: linear-gradient(90deg, #d4af37, #f59e0b);
          border-radius: 2px;
          transition: width 0.3s ease;
        }
      }

      .loading-detail {
        font-size: 11px;
        color: #64748b;
      }
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }
  }

  .three-canvas {
    width: 100% !important;
    height: 100% !important;
    display: block;
    cursor: grab;
    &:active { cursor: grabbing; }
  }

  .controls-panel {
    position: absolute;
    top: 12px;
    left: 12px;
    z-index: 10;
    background: rgba(15,23,42,0.85);
    backdrop-filter: blur(8px);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 12px;

    .control-group {
      margin-bottom: 10px;
      &:last-child { margin-bottom: 0; }
    }
    .control-label {
      font-size: 11px;
      color: var(--text-muted);
      margin-bottom: 6px;
      letter-spacing: 1px;
    }
    :deep(.el-checkbox) { margin-right: 10px; color: var(--text-secondary); }
  }

  .info-panel {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 10;
    min-width: 220px;
    background: rgba(15,23,42,0.85);
    backdrop-filter: blur(8px);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 12px 14px;

    .info-title {
      display: flex; align-items: center; gap: 8px;
      font-size: 15px; font-weight: 600; color: var(--accent-gold);
      padding-bottom: 8px; margin-bottom: 8px;
      border-bottom: 1px solid var(--border-color);
    }
    .info-row {
      display: flex; justify-content: space-between;
      font-size: 12px; padding: 3px 0; color: var(--text-secondary);
      b { color: var(--text-primary); font-weight: 600; &.accent { color: var(--accent-gold); } &.danger { color: var(--accent-red); } }
    }
  }

  .stats-legend {
    position: absolute;
    bottom: 12px;
    left: 12px;
    z-index: 10;
    display: flex; gap: 14px;
    background: rgba(15,23,42,0.85);
    backdrop-filter: blur(8px);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 8px 14px;
    font-size: 12px; color: var(--text-secondary);

    .legend-item {
      display: flex; align-items: center; gap: 6px;
      .dot {
        width: 10px; height: 10px; border-radius: 50%;
        box-shadow: 0 0 8px currentColor;
        &.normal { background: #10b981; color: #10b981; }
        &.warning { background: #f59e0b; color: #f59e0b; }
        &.danger { background: #ef4444; color: #ef4444; animation: glow 1.5s infinite; }
      }
    }
    @keyframes glow {
      0%,100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  }
}
</style>
