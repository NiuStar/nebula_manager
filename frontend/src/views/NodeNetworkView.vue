<template>
  <div class="network">
    <section class="card">
      <div class="header">
        <button class="btn secondary" type="button" @click="goBack">返回</button>
        <div class="title">
          <h2>网络情况 - {{ state.data?.node.name || '加载中' }}</h2>
          <p v-if="state.data" class="muted">
            子网 IP：{{ state.data.node.subnet_ip || '未配置' }}
          </p>
        </div>
      </div>

      <div class="controls">
        <span>时间范围：</span>
        <div class="range-group">
          <button
            v-for="option in ranges"
            :key="option.value"
            class="btn secondary"
            :class="{ active: state.range === option.value }"
            type="button"
            @click="setRange(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div class="chart-area">
        <div v-if="state.loading" class="placeholder">正在加载数据…</div>
        <div v-else-if="state.error" class="placeholder error">{{ state.error }}</div>
        <div v-else-if="!state.data || !state.data.peers.length" class="placeholder">暂无其它节点。</div>
        <div v-else-if="!chartModel.series.length" class="placeholder">暂无可绘制的延迟数据。</div>
        <div v-else class="chart-wrapper">
          <svg
            class="chart"
            :viewBox="`0 0 ${chartModel.width} ${chartModel.height}`"
            preserveAspectRatio="none"
          >
            <g class="axes">
              <line
                class="axis"
                :x1="chartModel.padding.left"
                :y1="chartModel.height - chartModel.padding.bottom"
                :x2="chartModel.width - chartModel.padding.right"
                :y2="chartModel.height - chartModel.padding.bottom"
              />
              <line
                class="axis"
                :x1="chartModel.padding.left"
                :y1="chartModel.padding.top"
                :x2="chartModel.padding.left"
                :y2="chartModel.height - chartModel.padding.bottom"
              />

              <g v-for="tick in chartModel.xTicks" :key="`x-${tick.x}`" class="tick x">
                <line
                  :x1="tick.x"
                  :y1="chartModel.height - chartModel.padding.bottom"
                  :x2="tick.x"
                  :y2="chartModel.height - chartModel.padding.bottom + 6"
                />
                <text :x="tick.x" :y="chartModel.height - chartModel.padding.bottom + 22">{{ tick.label }}</text>
              </g>

              <g v-for="tick in chartModel.yTicks" :key="`y-${tick.y}`" class="tick y">
                <line
                  :x1="chartModel.padding.left - 6"
                  :y1="tick.y"
                  :x2="chartModel.width - chartModel.padding.right"
                  :y2="tick.y"
                  class="grid"
                />
                <text :x="chartModel.padding.left - 10" :y="tick.y + 4">{{ tick.label }}</text>
              </g>
            </g>

            <g class="series">
              <g v-for="series in chartModel.series" :key="series.label">
                <polyline :points="series.path" :stroke="series.color" fill="none" stroke-width="2" />
                <circle
                  v-for="(point, idx) in series.points"
                  :key="`${series.label}-${idx}`"
                  :cx="point.cx"
                  :cy="point.cy"
                  :fill="series.color"
                  r="3"
                />
              </g>
            </g>
          </svg>

          <div class="legend" v-if="chartLegend.length">
            <span
              v-for="legend in chartLegend"
              :key="legend.label"
              :class="{ muted: !legend.hasData }"
            >
              <span class="dot" :style="{ backgroundColor: legend.color }"></span>
              {{ legend.label }}<span v-if="!legend.hasData">（暂无数据）</span>
            </span>
          </div>
        </div>
      </div>
    </section>

    <section class="card">
      <h3>节点联通情况</h3>
      <table class="table">
        <thead>
          <tr>
            <th>目标节点</th>
            <th>子网 IP</th>
            <th>最近延迟 (ms)</th>
            <th>最后采样时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="peer in peerSummaries" :key="peer.id">
            <td>{{ peer.name }}</td>
            <td>{{ peer.subnet_ip || '-' }}</td>
            <td :class="{ failed: peer.last?.success === false }">
              <span v-if="peer.last">
                {{ peer.last.success ? peer.last.latency_ms.toFixed(2) : '失败' }}
              </span>
              <span v-else>-</span>
            </td>
            <td>
              <span v-if="peer.last">{{ formatTime(peer.last.timestamp) }}</span>
              <span v-else>-</span>
            </td>
          </tr>
          <tr v-if="!peerSummaries.length">
            <td colspan="4">暂无其它节点。</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getNodeNetwork, getPublicNodeNetwork } from '../api';

const router = useRouter();
const route = useRoute();

const ranges = [
  { label: '最近 1 小时', value: '1h' },
  { label: '最近 6 小时', value: '6h' },
  { label: '最近 24 小时', value: '24h' }
];

const state = reactive({
  range: '1h',
  loading: false,
  error: '',
  data: null
});
const isPublic = computed(() => route.name === 'public-node-network');

const palette = ['#2563eb', '#dc2626', '#10b981', '#f59e0b', '#9333ea', '#ec4899', '#0ea5e9', '#f97316'];
const spanLookup = {
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000
};
const REFRESH_INTERVAL = 60 * 1000;
let refreshTimer = null;

function goBack() {
  if (isPublic.value) {
    router.push({ name: 'public-status' });
  } else {
    router.push({ name: 'nodes' });
  }
}

function setRange(value) {
  if (state.range === value) {
    return;
  }
  state.range = value;
  restartRefreshTimer();
  fetchData();
}

function currentNodeId() {
  return Number(route.params.id);
}

async function fetchData() {
  const nodeId = currentNodeId();
  if (!nodeId) {
    state.error = '无效的节点编号';
    return;
  }
  state.loading = true;
  state.error = '';
  try {
    const request = isPublic.value ? getPublicNodeNetwork : getNodeNetwork;
    const { data } = await request(nodeId, state.range);
    state.data = data.data;
  } catch (err) {
    state.error = err.response?.data?.error || '数据加载失败';
  } finally {
    state.loading = false;
  }
}

function restartRefreshTimer() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
  refreshTimer = setInterval(() => {
    fetchData();
  }, REFRESH_INTERVAL);
}

function formatTime(value) {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

const chartModel = computed(() => {
  const base = {
    width: 860,
    height: 360,
    padding: { left: 60, right: 24, top: 20, bottom: 48 },
    series: [],
    xTicks: [],
    yTicks: []
  };

  const peers = state.data?.peers || [];
  if (!peers.length) {
    return base;
  }

  const span = spanLookup[state.range] || spanLookup['1h'];
  const seriesData = [];
  let minTime = Number.POSITIVE_INFINITY;
  let maxTime = Number.NEGATIVE_INFINITY;
  let maxValue = 0;

  peers.forEach((peer, index) => {
    const color = palette[index % palette.length];
    const raw = (peer.points || [])
      .filter((point) => point && point.success)
      .map((point) => {
        const time = new Date(point.timestamp).getTime();
        if (Number.isNaN(time)) {
          return null;
        }
        return { time, value: Number(point.latency_ms) };
      })
      .filter(Boolean)
      .sort((a, b) => a.time - b.time);

    if (raw.length) {
      minTime = Math.min(minTime, raw[0].time);
      maxTime = Math.max(maxTime, raw[raw.length - 1].time);
      raw.forEach((p) => {
        if (p.value > maxValue) {
          maxValue = p.value;
        }
      });
    }

    seriesData.push({ label: peer.peer.name, color, raw });
  });

  if (!Number.isFinite(minTime) || !Number.isFinite(maxTime)) {
    return base;
  }

  if (maxTime <= minTime) {
    maxTime = minTime + span;
  } else {
    maxTime += Math.max(60000, span * 0.05);
  }

  if (maxValue <= 0) {
    maxValue = 1;
  } else {
    maxValue *= 1.2;
  }

  const { width, height, padding } = base;
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  const projectPoint = (point) => {
    const xRatio = (point.time - minTime) / (maxTime - minTime);
    const yRatio = point.value / maxValue;
    const x = padding.left + xRatio * chartWidth;
    const y = padding.top + (1 - yRatio) * chartHeight;
    return { x, y };
  };

  const series = seriesData
    .filter((s) => s.raw.length)
    .map((s) => {
      const coords = s.raw.map(projectPoint);
      const path = coords.map((p) => `${p.x.toFixed(2)},${p.y.toFixed(2)}`).join(' ');
      const points = coords.map((p) => ({ cx: p.x, cy: p.y }));
      return { label: s.label, color: s.color, path, points };
    });

  if (!series.length) {
    return base;
  }

  const xTickCount = 4;
  const xTicks = Array.from({ length: xTickCount }, (_, idx) => {
    const ratio = idx / (xTickCount - 1);
    const time = minTime + ratio * (maxTime - minTime);
    const x = padding.left + ratio * chartWidth;
    return { x, label: formatTime(time) };
  });

  const yTickCount = 5;
  const yTicks = Array.from({ length: yTickCount }, (_, idx) => {
    const ratio = idx / (yTickCount - 1);
    const value = maxValue * (1 - ratio);
    const y = padding.top + ratio * chartHeight;
    return {
      y,
      label: value >= 100 ? value.toFixed(0) : value.toFixed(1)
    };
  });

  return { width, height, padding, series, xTicks, yTicks };
});

const chartLegend = computed(() => {
  if (!state.data?.peers?.length) {
    return [];
  }
  return state.data.peers.map((peer, index) => {
    const hasData = (peer.points || []).some((point) => point && point.success);
    return {
      label: peer.peer.name,
      color: palette[index % palette.length],
      hasData
    };
  });
});

const peerSummaries = computed(() => {
  if (!state.data?.peers?.length) {
    return [];
  }
  return state.data.peers.map((peer) => {
    const points = peer.points || [];
    const last = points.length ? points[points.length - 1] : null;
    return {
      id: peer.peer.id,
      name: peer.peer.name,
      subnet_ip: peer.peer.subnet_ip,
      last
    };
  });
});

onMounted(() => {
  if (route.query.range && ['1h', '6h', '24h'].includes(route.query.range)) {
    state.range = route.query.range;
  }
  fetchData();
  restartRefreshTimer();
});

watch(
  () => route.params.id,
  () => {
    restartRefreshTimer();
    fetchData();
  }
);

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<style scoped>
.network {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.header {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header .title {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.range-group {
  display: inline-flex;
  gap: 0.5rem;
}

.range-group .btn.active {
  background: #2563eb;
  color: #ffffff;
  border-color: transparent;
}

.chart-area {
  position: relative;
  min-height: 320px;
}


.chart-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.chart {
  width: 100%;
  height: 320px;
}

.axes .axis {
  stroke: #334155;
  stroke-width: 1;
}

.tick.x line {
  stroke: #334155;
  stroke-width: 1;
}

.tick.y .grid {
  stroke: rgba(148, 163, 184, 0.25);
  stroke-dasharray: 4 4;
}

.tick text {
  fill: #475569;
  font-size: 12px;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  font-size: 0.9rem;
  color: #0f172a;
}

.legend .dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 0.4rem;
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #64748b;
  height: 320px;
  background: rgba(148, 163, 184, 0.08);
  border-radius: 8px;
}

.placeholder.error {
  color: #dc2626;
}

.table .failed {
  color: #dc2626;
  font-weight: 600;
}

.muted {
  color: #64748b;
  font-size: 0.9rem;
}
</style>
