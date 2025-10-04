<template>
  <div class="nodes">
    <section class="card toolbar-card">
      <div class="toolbar">
        <div>
          <h2>节点概览</h2>
          <p class="muted">实时掌握机器健康情况与网络质量</p>
        </div>
        <div class="toolbar-actions">
          <div class="view-toggle">
            <button :class="{ active: viewMode === 'card' }" type="button" @click="setView('card')">卡片视图</button>
            <button :class="{ active: viewMode === 'table' }" type="button" @click="setView('table')">列表视图</button>
          </div>
          <button class="btn" type="button" @click="openCreateModal">创建节点</button>
        </div>
      </div>
    </section>

    <section v-if="viewMode === 'table'" class="card">
      <h2>节点列表</h2>
      <table class="table">
        <thead>
          <tr>
            <th>名称</th>
            <th>类型</th>
            <th>子网 IP</th>
            <th>端口</th>
            <th>下载代理</th>
            <th>安装命令</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="node in nodes" :key="node.id">
            <td>{{ node.name }}</td>
            <td>{{ renderRole(node.role) }}</td>
            <td>{{ node.subnet_ip }}</td>
            <td>{{ node.port }}</td>
            <td>{{ renderProxy(node.proxy_mode) }}</td>
            <td>
              <div class="command">
                <code>{{ node.install_command }}</code>
                <button class="btn secondary" type="button" @click="copyCommand(node.install_command)">复制</button>
              </div>
            </td>
            <td class="actions">
              <button class="btn secondary" type="button" @click="viewNetwork(node)">网络情况</button>
              <button class="btn secondary" type="button" @click="downloadBundle(node.id)">下载归档</button>
              <button class="btn danger" type="button" @click="removeNode(node)">删除</button>
            </td>
          </tr>
          <tr v-if="!nodes.length">
            <td colspan="8">暂无节点，请先在上方创建。</td>
          </tr>
        </tbody>
      </table>
    </section>

    <section v-if="viewMode === 'card' && nodes.length" class="card">
      <div class="status-header">
        <h2>节点运行状态</h2>
        <span class="muted">数据每分钟自动刷新</span>
      </div>
      <div class="status-grid">
        <article
          v-for="node in nodes"
          :key="`status-${node.id}`"
          class="status-card"
          role="button"
          tabindex="0"
          @click="viewNetwork(node)"
          @keyup.enter="viewNetwork(node)"
        >
          <header class="status-card__header">
            <div>
              <h3>{{ node.name }}</h3>
              <p class="status-subtitle">
                <span>{{ renderRole(node.role) }}</span>
                <span v-if="node.status?.reported_at"> · 更新于 {{ formatRelativeTime(node.status.reported_at) }}</span>
              </p>
            </div>
            <button class="btn tiny" type="button" @click.stop="copyCommand(node.install_command)">复制安装命令</button>
          </header>

          <div v-if="node.status" class="status-body">
            <div class="meter">
              <span class="label">CPU</span>
              <div class="bar">
                <span class="fill" :style="{ width: meterWidth(node.status.cpu_usage) }"></span>
              </div>
              <span class="value">{{ formatPercent(node.status.cpu_usage) }}</span>
            </div>
            <div class="meter">
              <span class="label">内存</span>
              <div class="bar">
                <span class="fill" :style="{ width: meterRatio(node.status.memory_used, node.status.memory_total) }"></span>
              </div>
              <span class="value">{{ formatBytes(node.status.memory_used) }} / {{ formatBytes(node.status.memory_total) }}</span>
            </div>
            <div class="meter">
              <span class="label">磁盘</span>
              <div class="bar">
                <span class="fill" :style="{ width: meterRatio(node.status.disk_used, node.status.disk_total) }"></span>
              </div>
              <span class="value">{{ formatBytes(node.status.disk_used) }} / {{ formatBytes(node.status.disk_total) }}</span>
            </div>
            <div class="meter">
              <span class="label">Swap</span>
              <div class="bar">
                <span class="fill" :style="{ width: meterRatio(node.status.swap_used, node.status.swap_total) }"></span>
              </div>
              <span class="value">{{ formatBytes(node.status.swap_used) }} / {{ formatBytes(node.status.swap_total) }}</span>
            </div>

            <div class="status-footer">
              <div>
                <span class="muted">平均负载</span>
                <div>{{ formatLoad(node.status.load1) }} / {{ formatLoad(node.status.load5) }} / {{ formatLoad(node.status.load15) }}</div>
              </div>
              <div>
                <span class="muted">网络</span>
                <div>
                  ↑ {{ formatRate(node.status.txRate) }} · ↓ {{ formatRate(node.status.rxRate) }}
                </div>
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

  <div v-if="showCreateModal" class="modal-backdrop" @click.self="closeCreateModal">
    <div class="modal">
      <div class="modal-header">
        <h3>创建节点</h3>
        <button class="modal-close" type="button" @click="closeCreateModal">×</button>
      </div>
      <p class="muted">请先生成根证书并配置全局参数；执行安装命令前，记得通过登录页面获取 token 并设置 <code>NEBULA_ACCESS_TOKEN</code>。</p>
      <form class="grid" @submit.prevent="submitCreate">
        <label>
          <span>节点名称</span>
          <input v-model="form.name" placeholder="例如：web-01" required />
        </label>
        <label>
          <span>节点类型</span>
          <select v-model="form.role">
            <option value="standard">普通节点</option>
            <option value="lighthouse">灯塔节点</option>
          </select>
        </label>
        <label>
          <span>Nebula 子网 IP</span>
          <input v-model="form.subnet_ip" placeholder="例如：10.10.0.11" required />
        </label>
        <label>
          <span>公网 IP / 域名</span>
          <input v-model="form.public_ip" placeholder="如需被其它节点访问，请填写公网地址" />
        </label>
        <label>
          <span>监听端口</span>
          <input type="number" v-model.number="form.port" min="0" />
        </label>
        <label>
          <span>标签（逗号分隔）</span>
          <input v-model="tags" placeholder="例如：prod,web" />
        </label>
        <label>
          <span>下载代理</span>
          <select v-model="form.proxy_mode">
            <option value="none">不使用代理</option>
            <option value="ipv4">使用 IPv4 代理</option>
            <option value="ipv6">使用 IPv6 代理</option>
          </select>
          <small class="muted">
            若目标主机无法直接访问 GitHub，可选择代理；IPv6 选项使用 <code>http://[240b:4009:25a:1801:0:8005:7a35:c192]:5000/https://proxy.529851.xyz/</code>。
          </small>
        </label>
        <div class="full">
          <button class="btn" type="submit">生成节点资料</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { createNode, deleteNode, downloadNodeBundle, listNodes } from '../api';

const nodes = ref([]);
const tags = ref('');
const router = useRouter();
const statusSnapshots = new Map();
const viewMode = ref('card');
const showCreateModal = ref(false);
let refreshTimer = null;
const REFRESH_INTERVAL = 60 * 1000;

const form = reactive({
  name: '',
  role: 'standard',
  subnet_ip: '',
  public_ip: '',
  port: 0,
  proxy_mode: 'none'
});

const resetForm = () => {
  Object.assign(form, { name: '', role: 'standard', subnet_ip: '', public_ip: '', port: 0, proxy_mode: 'none' });
  tags.value = '';
};

function renderRole(role) {
  return role === 'lighthouse' ? '灯塔节点' : '普通节点';
}

function renderProxy(mode) {
  switch (mode) {
    case 'ipv4':
      return 'IPv4 代理';
    case 'ipv6':
      return 'IPv6 代理';
    default:
      return '无';
  }
}

async function fetchNodes() {
  try {
    const { data } = await listNodes();
    const list = data.data || [];
    enrichStatuses(list);
    nodes.value = list;
  } catch (err) {
    console.error(err);
  }
}

function parseTags() {
  return tags.value
    ? tags.value
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
    : [];
}

async function submitCreate() {
  try {
    const payload = JSON.parse(JSON.stringify(form));
    payload.tags = parseTags();
    if (payload.proxy_mode === 'none') {
      payload.proxy_mode = '';
    }
    await createNode(payload);
    window.alert('节点创建成功，可下载部署脚本进行安装');
    resetForm();
    closeCreateModal();
    fetchNodes();
  } catch (err) {
    window.alert(err.response?.data?.error || '节点创建失败');
  }
}

async function downloadBundle(id) {
  try {
    const response = await downloadNodeBundle(id);
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const link = document.createElement('a');
    link.href = url;
    link.download = `nebula-node-${id}.tar.gz`;
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (err) {
    window.alert('归档下载失败');
  }
}

async function copyCommand(command) {
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(command);
    } else {
      const textarea = document.createElement('textarea');
      textarea.value = command;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
    }
    window.alert('安装命令已复制，直接在目标主机执行即可');
  } catch (err) {
    window.prompt('复制失败，请手动复制命令：', command);
  }
}

async function removeNode(node) {
  if (!window.confirm(`确定要删除节点 ${node.name} 吗？该操作会移除数据库记录与生成的文件。`)) {
    return;
  }
  try {
    await deleteNode(node.id);
    window.alert('节点已删除');
    fetchNodes();
  } catch (err) {
    window.alert(err.response?.data?.error || '删除失败');
  }
}

function viewNetwork(node) {
  router.push({ name: 'node-network', params: { id: node.id } });
}

function enrichStatuses(list) {
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

function startAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
  refreshTimer = setInterval(fetchNodes, REFRESH_INTERVAL);
}

function setView(mode) {
  viewMode.value = mode;
}

function openCreateModal() {
  resetForm();
  showCreateModal.value = true;
}

function closeCreateModal() {
  showCreateModal.value = false;
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

onMounted(() => {
  fetchNodes();
  startAutoRefresh();
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
});
</script>

<style scoped>
.nodes {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  width: 100%;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

label {
  display: flex;
  flex-direction: column;
  font-weight: 600;
  gap: 0.35rem;
}

.full {
  grid-column: 1 / -1;
}

.actions {
  display: flex;
  gap: 0.4rem;
}

.actions .danger {
  background: #ef4444;
  border: none;
}

.actions .danger:hover {
  background: #dc2626;
}

.command {
  display: flex;
  gap: 0.6rem;
  align-items: center;
}

.command code {
  flex: 1;
  background: #0f172a;
  color: #e2e8f0;
  padding: 0.4rem 0.6rem;
  border-radius: 4px;
  overflow-x: auto;
}

.muted {
  color: #64748b;
  margin-bottom: 0.6rem;
}

.btn.tiny {
  padding: 0.2rem 0.6rem;
  font-size: 0.8rem;
  line-height: 1.2;
}

.status-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
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
  background: rgba(15, 23, 42, 0.78);
  border-radius: 12px;
  padding: 1.1rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.18);
  box-shadow: 0 18px 35px rgba(15, 23, 42, 0.25);
  cursor: pointer;
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.status-card:hover {
  transform: translateY(-4px);
  border-color: rgba(148, 163, 184, 0.35);
}

.toolbar-card {
  margin-bottom: 1rem;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.view-toggle {
  display: inline-flex;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 999px;
  overflow: hidden;
}

.view-toggle button {
  padding: 0.45rem 1rem;
  background: transparent;
  border: none;
  color: #cbd5f5;
  font-size: 0.85rem;
  cursor: pointer;
}

.view-toggle button.active {
  background: linear-gradient(120deg, rgba(59, 130, 246, 0.85), rgba(56, 189, 248, 0.85));
  color: #fff;
}

.status-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
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

.status-card .btn.tiny {
  padding: 0.25rem 0.6rem;
  font-size: 0.8rem;
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

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 40;
  padding: 1rem;
}

.modal {
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(148, 163, 184, 0.25);
  border-radius: 12px;
  width: min(620px, 100%);
  padding: 1.8rem;
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.25);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.2rem;
}

.modal h3 {
  margin: 0;
  font-size: 1.25rem;
}

.modal-close {
  background: none;
  border: none;
  color: #94a3b8;
  font-size: 1.2rem;
  cursor: pointer;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.full {
  grid-column: 1 / -1;
}
</style>
.install-command {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.2);
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
}

.install-command code {
  flex: 1;
  font-family: 'JetBrains Mono', 'Fira Code', Consolas, monospace;
  font-size: 0.78rem;
  color: #e2e8f0;
  word-break: break-all;
}

.install-command .btn.tiny {
  white-space: nowrap;
}
