<template>
  <div class="heatmap-container" ref="containerRef">
    <canvas ref="canvasRef" class="heatmap-canvas" />
    <svg ref="svgRef" class="heatmap-overlay">
      <defs>
        <radialGradient id="heatGradient" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="rgba(239,68,68,0.9)" />
          <stop offset="40%" stop-color="rgba(245,158,11,0.7)" />
          <stop offset="70%" stop-color="rgba(59,130,246,0.5)" />
          <stop offset="100%" stop-color="rgba(16,185,129,0.0)" />
        </radialGradient>
        <filter id="glow">
          <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>

      <g v-if="heatmapZones.length > 0" class="heatmap-zones">
        <circle
          v-for="(zone, i) in heatmapZones"
          :key="'zone-'+i"
          :cx="zone.x"
          :cy="zone.y"
          :r="zone.radius"
          fill="url(#heatGradient)"
          :opacity="zone.opacity"
          filter="url(#glow)"
        />
      </g>

      <g class="crack-lines">
        <line
          v-for="(crack, i) in crackLines"
          :key="'crack-'+i"
          :x1="crack.x1" :y1="crack.y1"
          :x2="crack.x2" :y2="crack.y2"
          :stroke="crack.color"
          :stroke-width="crack.width"
          stroke-linecap="round"
          filter="url(#glow)"
        />
        <g v-for="(crack, i) in crackLines" :key="'crack-pts-'+i">
          <circle
            v-for="(p, j) in interpolateCrack(crack)"
            :key="j"
            :cx="p.x" :cy="p.y" :r="1.5"
            :fill="crack.color"
            :opacity="0.8"
          />
        </g>
      </g>

      <g class="monitoring-points">
        <g
          v-for="pt in displayPoints"
          :key="pt.id"
          class="point-group"
          :transform="`translate(${pt._x}, ${pt._y})`"
          @click="$emit('point-click', pt.id)"
        >
          <circle
            v-if="pt._status === 'danger'"
            r="16" fill="none"
            stroke="#ef4444" stroke-width="2"
            opacity="0.6"
          >
            <animate attributeName="r" values="12;20;12" dur="1.5s" repeatCount="indefinite"/>
            <animate attributeName="opacity" values="0.8;0.2;0.8" dur="1.5s" repeatCount="indefinite"/>
          </circle>
          <circle
            :r="pt._status === 'danger' ? 10 : 8"
            :fill="statusColors[pt._status].bg"
            :stroke="statusColors[pt._status].border"
            stroke-width="2"
            class="point-marker"
            :filter="pt._status === 'danger' ? 'url(#glow)' : ''"
          />
          <text
            y="3" text-anchor="middle"
            fill="#fff" font-size="10" font-weight="bold"
            style="pointer-events:none"
          >{{ pt.id }}</text>
          <title>{{ pt.name }} | 硬度: {{ pt._hardness }} | 裂隙: {{ pt._crack }}mm</title>
        </g>
      </g>
    </svg>

    <div class="canvas-legend">
      <div class="legend-title">风化强度</div>
      <div class="legend-bar"></div>
      <div class="legend-labels">
        <span>低</span><span>中</span><span>高</span>
      </div>
    </div>
  </div>
</template>

<script setup>import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue';
const props = defineProps({
 cave: Object,
 points: { type: Array, default: () => [] },
 sensorData: { type: Array, default: () => [] },
 width: { type: Number, default: 0 },
 height: { type: Number, default: 0 }
});
defineEmits(['point-click']);
const containerRef = ref(null);
const canvasRef = ref(null);
const svgRef = ref(null);
const dims = ref({ w: 800, h: 420 });
const statusColors = {
 normal: { bg: '#10b981', border: '#065f46' },
 warning: { bg: '#f59e0b', border: '#92400e' },
 danger: { bg: '#ef4444', border: '#991b1b' },
 unknown: { bg: '#64748b', border: '#334155' }
};
const displayPoints = computed(() => {
 const { w, h } = dims.value;
 const padding = 60;
 let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
 props.points.forEach(p => {
 minX = Math.min(minX, p.positionX);
 maxX = Math.max(maxX, p.positionX);
 minY = Math.min(minY, p.positionY);
 maxY = Math.max(maxY, p.positionY);
 });
 if (!isFinite(minX)) {
 minX = -50; maxX = 50; minY = -50; maxY = 50;
 }
 const rangeX = maxX - minX || 100;
 const rangeY = maxY - minY || 100;
 return props.points.map(pt => {
 const nx = (pt.positionX - minX) / rangeX;
 const ny = (pt.positionY - minY) / rangeY;
 const sensor = props.sensorData.find(s => s.pointId === pt.id);
 const hardness = sensor?.surfaceHardness ?? pt.initialHardness;
 const crack = sensor?.crackWidth ?? pt.initialCrackWidth;
 let status = 'normal';
 const dropRatio = pt.initialHardness > 0 ? (pt.initialHardness - hardness) / pt.initialHardness * 100 : 0;
 if (crack > 0.5 || dropRatio > 20)
 status = 'danger';
 else if (crack > 0.3 || dropRatio > 12)
 status = 'warning';
 else if (!sensor)
 status = 'unknown';
 return {
 ...pt,
 _x: padding + nx * (w - 2 * padding),
 _y: padding + ny * (h - 2 * padding),
 _hardness: hardness?.toFixed?.(1) || hardness,
 _crack: crack?.toFixed?.(3) || crack,
 _status: status,
 _dropRatio: dropRatio,
 _weatheringScore: (dropRatio * 3 + crack * 80) / 4
 };
 });
});
const heatmapZones = computed(() => {
 return displayPoints.value
 .filter(p => p._weatheringScore > 3)
 .map(p => {
 const score = Math.min(100, p._weatheringScore);
 return {
 x: p._x,
 y: p._y,
 radius: 25 + score * 1.2,
 opacity: Math.min(0.85, 0.3 + score / 100)
 };
 })
 .sort((a, b) => a.opacity - b.opacity);
});
const crackLines = computed(() => {
 const lines = [];
 displayPoints.value.forEach(pt => {
 const sensor = props.sensorData.find(s => s.pointId === pt.id);
 const crackWidth = sensor?.crackWidth ?? pt.initialCrackWidth;
 if (crackWidth > 0.05) {
 const severity = Math.min(1, crackWidth / 0.8);
 const color = severity > 0.7 ? '#ef4444' : severity > 0.4 ? '#f59e0b' : '#3b82f6';
 const length = 15 + crackWidth * 60;
 const angle = ((pt.id * 137.5) % 360) * Math.PI / 180;
 const x2 = pt._x + Math.cos(angle) * length;
 const y2 = pt._y + Math.sin(angle) * length * 0.6;
 lines.push({
 x1: pt._x,
 y1: pt._y,
 x2, y2,
 color,
 width: 1 + severity * 3
 });
 if (crackWidth > 0.2) {
 const angle2 = angle + Math.PI / 3;
 const x2b = pt._x + Math.cos(angle2) * length * 0.6;
 const y2b = pt._y + Math.sin(angle2) * length * 0.4;
 lines.push({
 x1: pt._x,
 y1: pt._y,
 x2: x2b,
 y2: y2b,
 color,
 width: 1 + severity * 2
 });
 }
 }
 });
 return lines;
});
function interpolateCrack(crack) {
 const pts = [];
 const steps = 8;
 for (let i = 1; i < steps; i++) {
 const t = i / steps;
 const noise = (Math.sin(i * 12.9898 + crack.x1) * 43758.5453) % 1;
 pts.push({
 x: crack.x1 + (crack.x2 - crack.x1) * t + (noise - 0.5) * 6,
 y: crack.y1 + (crack.y2 - crack.y1) * t + (noise - 0.5) * 4
 });
 }
 return pts;
}
function drawCaveBackground() {
 const canvas = canvasRef.value;
 if (!canvas)
 return;
 const ctx = canvas.getContext('2d');
 const { w, h } = dims.value;
 canvas.width = w * devicePixelRatio;
 canvas.height = h * devicePixelRatio;
 canvas.style.width = w + 'px';
 canvas.style.height = h + 'px';
 ctx.scale(devicePixelRatio, devicePixelRatio);
 ctx.clearRect(0, 0, w, h);
 const grad = ctx.createRadialGradient(w / 2, h * 0.55, 50, w / 2, h * 0.5, w * 0.7);
 grad.addColorStop(0, '#1e293b');
 grad.addColorStop(0.5, '#0f172a');
 grad.addColorStop(1, '#020617');
 ctx.fillStyle = grad;
 ctx.fillRect(0, 0, w, h);
 ctx.strokeStyle = 'rgba(245,158,11,0.05)';
 ctx.lineWidth = 1;
 for (let x = 0; x <= w; x += 40) {
 ctx.beginPath();
 ctx.moveTo(x, 0);
 ctx.lineTo(x, h);
 ctx.stroke();
 }
 for (let y = 0; y <= h; y += 40) {
 ctx.beginPath();
 ctx.moveTo(0, y);
 ctx.lineTo(w, y);
 ctx.stroke();
 }
 drawCaveContours(ctx, w, h);
}
function drawCaveContours(ctx, w, h) {
 ctx.save();
 ctx.strokeStyle = 'rgba(148,163,184,0.15)';
 ctx.fillStyle = 'rgba(245,158,11,0.03)';
 ctx.lineWidth = 1.5;
 const cx = w / 2, cy = h / 2;
 const layers = [
 { rx: w * 0.42, ry: h * 0.40, offset: 0 },
 { rx: w * 0.35, ry: h * 0.33, offset: 0.02 },
 { rx: w * 0.28, ry: h * 0.26, offset: 0.04 },
 { rx: w * 0.20, ry: h * 0.18, offset: 0.06 }
 ];
 layers.forEach((layer, idx) => {
 ctx.beginPath();
 for (let a = 0; a <= Math.PI * 2.01; a += 0.05) {
 const noise = Math.sin(a * (3 + idx)) * 8 + Math.cos(a * (7 - idx)) * 4;
 const rpx = layer.rx + noise;
 const rpy = layer.ry + noise * 0.7;
 const x = cx + Math.cos(a) * rpx;
 const y = cy + Math.sin(a) * rpy;
 if (a === 0)
 ctx.moveTo(x, y);
 else
 ctx.lineTo(x, y);
 }
 ctx.closePath();
 ctx.globalAlpha = 0.4 - layer.offset * 3;
 ctx.fill();
 ctx.globalAlpha = 1;
 ctx.stroke();
 });
 const niches = [
 { x: 0.30, y: 0.35, w: 0.08, h: 0.12 },
 { x: 0.55, y: 0.28, w: 0.09, h: 0.14 },
 { x: 0.72, y: 0.42, w: 0.07, h: 0.10 },
 { x: 0.40, y: 0.62, w: 0.10, h: 0.13 },
 { x: 0.62, y: 0.68, w: 0.08, h: 0.11 },
 { x: 0.20, y: 0.58, w: 0.06, h: 0.09 }
 ];
 niches.forEach(n => {
 const nx = n.x * w, ny = n.y * h, nw = n.w * w, nh = n.h * h;
 ctx.beginPath();
 ctx.ellipse(nx + nw / 2, ny + nh / 2, nw / 2, nh / 2, 0, 0, Math.PI * 2);
 ctx.fillStyle = 'rgba(0,0,0,0.35)';
 ctx.fill();
 ctx.strokeStyle = 'rgba(245,158,11,0.2)';
 ctx.lineWidth = 1;
 ctx.stroke();
 });
 ctx.restore();
}
function resizeObserver() {
 if (!containerRef.value)
 return;
 const ro = new ResizeObserver(entries => {
 for (const entry of entries) {
 const cr = entry.contentRect;
 dims.value = {
 w: Math.floor(cr.width) || 800,
 h: Math.floor(cr.height) || 420
 };
 nextTick(() => drawCaveBackground());
 }
 });
 ro.observe(containerRef.value);
 return ro;
}
let observer = null;
watch(dims, () => {
 nextTick(() => drawCaveBackground());
});
onMounted(() => {
 if (containerRef.value) {
 const rect = containerRef.value.getBoundingClientRect();
 dims.value = {
 w: Math.floor(rect.width) || 800,
 h: Math.floor(rect.height) || 420
 };
 }
 nextTick(() => drawCaveBackground());
 observer = resizeObserver();
});
onBeforeUnmount(() => {
 observer?.disconnect?.();
});
</script>

<style lang="scss" scoped>
.heatmap-container {
  position: relative;
  width: 100%;
  height: 100%;
  border-radius: 8px;
  overflow: hidden;
  background: #0a0f1a;

  .heatmap-canvas {
    position: absolute; inset: 0;
    width: 100%; height: 100%;
    z-index: 1;
  }

  .heatmap-overlay {
    position: absolute; inset: 0;
    width: 100%; height: 100%;
    z-index: 2;

    .point-group {
      cursor: pointer;
      .point-marker {
        transition: r 0.2s ease;
        &:hover { r: 12; }
      }
    }
  }

  .canvas-legend {
    position: absolute;
    left: 12px;
    bottom: 12px;
    z-index: 3;
    background: rgba(15,23,42,0.7);
    border: 1px solid var(--border-color);
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 11px;

    .legend-title {
      color: var(--text-secondary);
      margin-bottom: 4px;
      font-size: 11px;
    }
    .legend-bar {
      width: 140px;
      height: 8px;
      border-radius: 4px;
      background: linear-gradient(90deg, #10b981, #3b82f6, #f59e0b, #ef4444);
    }
    .legend-labels {
      display: flex;
      justify-content: space-between;
      margin-top: 4px;
      color: var(--text-muted);
      font-size: 10px;
    }
  }
}
</style>
