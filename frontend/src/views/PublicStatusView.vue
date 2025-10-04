<template>
  <div class="public-status">
    <section class="hero">
      <h1>节点运行状态</h1>
      <p class="muted">实时展示各节点的 CPU、内存、磁盘与网络情况</p>
    </section>

    <section class="card">
      <div class="status-grid">
        <article v-for="node in nodes" :key="node.id" class="status-card">
          <header class="status-card__header">
            <div>
              <h3>{{ node.name }}</h3>
              <p class="status-subtitle">
                <span>{{ renderRole(node.role) }}</span>
                <span v-if="node.status?.reported_at"> · 更新于 {{ formatRelativeTime(node.status.reported_at) }}</span>
              </p>
            </div>
          </header>

          <div v-if="node.status" class="status-body">
            <div class="meter">
              <span class="label">CPU</span>
              <div class="bar"><span class="fill" :style="{ width: meterWidth(node.status.cpu_usage) }"></span></div>
              <span class="value">{{ formatPercent(node.status.cpu_usage) }}</span>
            </div>
            <div class="meter">
              <span class="label">内存</span>
              <div class="bar"><span class="fill" :style="{ width: meterRatio(node.status.memory_used, node.status.memory_total) }"></span></div>
              <span class="value">{{ formatBytes(node.status.memory_used) }} / {{ formatBytes(node.status.memory_total) }}</span>
            </div>
            <div class="meter">
              <span class="label">磁盘</span>
              <div class="bar"><span class="fill" :style="{ width: meterRatio(node.status.disk_used, node.status.disk_total) }"></span></div>
              <span class="value">{{ formatBytes(node.status.disk_used) }} / {{ formatBytes(node.status.disk_total) }}</span>
            </div>
            <div class="meter">
              <span class="label">Swap</span>
              <div class="bar"><span class="fill" :style="{ width: meterRatio(node.status.swap_used, node.status.swap_total) }"></span></div>
              <span class="value">{{ formatBytes(node.status.swap_used) }} / {{ formatBytes(node.status.swap_total) }}</span>
            </div>

            <div class="status-footer">
              <div>
                <span class="muted">平均负载</span>
                <div>{{ formatLoad(node.status.load1) }} / {{ formatLoad(node.status.load5) }} / {{ formatLoad(node.status.load15) }}</div>
              </div>
              <div>
                <span class="muted">网络</span>
                <div>↑ {{ formatRate(node.status.txRate) }} · ↓ {{ formatRate(node.status.rxRate) }}</div>
              </div>
              <div>
                <span class="muted">进程</span>
                <div>{{ node.status.processes || '-' }}</div>
              </div>
              <div>
                <span class="muted">运行时长</span>
                <div>{{ formatUptime(node.status.uptime) }}</div>
              </div>
            </div>
          </div>
          <div v-else class="status-empty">暂未上报运行状态</div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue';
import { getPublicStatus } from '../api';

const nodes = ref([]);
const REFRESH_INTERVAL = 60 * 1000;
let refreshTimer = null;
const statusSnapshots = new Map();

function renderRole(role) {
  return role === 'lighthouse' ? '灯塔节点' : '普通节点';
}

async function fetchStatus() {
  try {
    const { data } = await getPublicStatus();
    const list = data.data || [];
    enrich(list);
    nodes.value = list;
  } catch (err) {
    console.error('加载节点状态失败', err);
  }
}

function enrich(list) {
  const now = Date.now();
  const seen = new Set();
  list.forEach((node) => {
    const status = node.status;
    if (!status) {
      statusSnapshots.delete(node.id);
      return;
    }
    seen.add(node.id);
    const prev = statusSnapshots.get(node.id);
    if (prev && prev.rx <= status.net_rx_bytes && prev.tx <= status.net_tx_bytes) {
      const elapsed = Math.max((now - prev.timestamp) / 1000, 1);
      status.rxRate = (status.net_rx_bytes - prev.rx) / elapsed;
      status.txRate = (status.net_tx_bytes - prev.tx) / elapsed;
    } else {
      status.rxRate = null;
      status.txRate = null;
    }
    statusSnapshots.set(node.id, {
      rx: status.net_rx_bytes,
      tx: status.net_tx_bytes,
      timestamp: now
    });
  });
  Array.from(statusSnapshots.keys()).forEach((id) => {
    if (!seen.has(id)) {
      statusSnapshots.delete(id);
    }
  });
}

function meterWidth(value) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '0%';
  }
  const safe = Math.max(0, Math.min(100, value));
  return `${safe.toFixed(1)}%`;
}

function meterRatio(used, total) {
  if (!total || total <= 0) {
    return '0%';
  }
  const val = Math.max(0, Math.min(1, used / total));
  return `${(val * 100).toFixed(1)}%`;
}

function formatBytes(value) {
  if (!value || value <= 0) {
    return '0B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let idx = 0;
  let num = value;
  while (num >= 1024 && idx < units.length - 1) {
    num /= 1024;
    idx += 1;
  }
  return `${num.toFixed(idx === 0 ? 0 : 1)}${units[idx]}`;
}

function formatRate(value) {
  if (value === null || value === undefined) {
    return '-';
  }
  return `${formatBytes(Math.max(value, 0))}/s`;
}

function formatPercent(value) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return `${value.toFixed(1)}%`;
}

function formatLoad(value) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return value.toFixed(2);
}

function formatUptime(seconds) {
  if (!seconds || seconds <= 0) {
    return '-';
  }
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) {
    return `${days}天 ${hours}小时`;
  }
  if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`;
  }
  return `${minutes}分钟`;
}

function formatRelativeTime(iso) {
  if (!iso) {
    return '-';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  const diff = Date.now() - date.getTime();
  if (diff < 0) {
    return '刚刚';
  }
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) {
    return '刚刚';
  }
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} 小时前`;
  }
  const days = Math.floor(hours / 24);
  return `${days} 天前`;
}

function startAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
  refreshTimer = setInterval(fetchStatus, REFRESH_INTERVAL);
}

onMounted(() => {
  fetchStatus();
  startAutoRefresh();
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
});
</script>

<style scoped>
.public-status {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  padding: 1rem;
}

.hero {
  text-align: center;
  padding: 2rem 1rem;
}

.hero h1 {
  margin: 0;
  font-size: 2rem;
}

.muted {
  color: #64748b;
}

.card {
  background: rgba(15, 23, 42, 0.92);
  border-radius: 16px;
  padding: 1.8rem;
  border: 1px solid rgba(148, 163, 184, 0.25);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.25);
}

.status-grid {
  display: grid;
  gap: 1.2rem;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
}

@media (max-width: 768px) {
  .status-grid {
    grid-template-columns: 1fr;
  }
}

@media (min-width: 1024px) {
  .status-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

.status-card {
  background: rgba(15, 23, 42, 0.8);
  border-radius: 12px;
  padding: 1.2rem 1.4rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.18);
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.06);
}

.status-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-card h3 {
  margin: 0;
  font-size: 1.1rem;
  color: #f8fafc;
}

.status-subtitle {
  color: #cbd5f5;
  font-size: 0.85rem;
  margin: 0.15rem 0 0;
}

.status-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.meter {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.meter .label {
  width: 3.5rem;
  color: rgba(229, 231, 235, 0.9);
  font-size: 0.85rem;
}

.meter .bar {
  flex: 1;
  height: 8px;
  background: rgba(148, 163, 184, 0.25);
  border-radius: 999px;
  position: relative;
  overflow: hidden;
}

.meter .fill {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  background: linear-gradient(90deg, rgba(56, 189, 248, 0.9), rgba(59, 130, 246, 0.95));
  border-radius: 999px;
}

.meter .value {
  min-width: 9rem;
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: #f1f5f9;
  font-size: 0.85rem;
}

.status-footer {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem 1rem;
  font-size: 0.85rem;
  color: rgba(226, 232, 240, 0.95);
}

.status-empty {
  color: rgba(203, 213, 225, 0.9);
  text-align: center;
  padding: 1.5rem 0;
  border: 1px dashed rgba(148, 163, 184, 0.4);
  border-radius: 8px;
}
</style>
