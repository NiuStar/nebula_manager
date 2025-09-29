<template>
  <div class="nodes">
    <section class="card">
      <h2>创建节点</h2>
      <p class="muted">请先生成根证书并配置全局参数；执行安装命令前，记得通过登录页面获取 token 并设置 <code>NEBULA_ACCESS_TOKEN</code>。</p>
      <form class="grid" @submit.prevent="create">
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
    </section>

    <section class="card">
      <h2>已管理的节点</h2>
      <table class="table">
        <thead>
          <tr>
            <th>名称</th>
            <th>类型</th>
            <th>子网 IP</th>
            <th>公网地址</th>
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
            <td>{{ node.public_ip || '-' }}</td>
            <td>{{ node.port }}</td>
            <td>{{ renderProxy(node.proxy_mode) }}</td>
            <td>
              <div class="command">
                <code>{{ node.install_command }}</code>
                <button class="btn secondary" type="button" @click="copyCommand(node.install_command)">复制</button>
              </div>
            </td>
            <td class="actions">
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
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue';
import { createNode, deleteNode, downloadNodeBundle, listNodes } from '../api';

const nodes = ref([]);
const tags = ref('');

const form = reactive({
  name: '',
  role: 'standard',
  subnet_ip: '',
  public_ip: '',
  port: 0,
  proxy_mode: 'none'
});

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
    nodes.value = data.data || [];
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

async function create() {
  try {
    const payload = JSON.parse(JSON.stringify(form));
    payload.tags = parseTags();
    if (payload.proxy_mode === 'none') {
      payload.proxy_mode = '';
    }
    await createNode(payload);
    window.alert('节点创建成功，可下载部署脚本进行安装');
    Object.assign(form, { name: '', role: 'standard', subnet_ip: '', public_ip: '', port: 0, proxy_mode: 'none' });
    tags.value = '';
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

onMounted(fetchNodes);
</script>

<style scoped>
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
</style>
