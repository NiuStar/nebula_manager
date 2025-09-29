<template>
  <div class="dashboard">
    <section class="card">
      <h2>根证书管理</h2>
      <p v-if="ca">
        当前根证书：<strong>{{ ca.name }}</strong>
        <span class="muted">（创建时间：{{ formatDate(ca.created_at) }}）</span>
      </p>
      <p v-else>尚未生成根证书，请先完成下面的表单。</p>

      <form class="form" @submit.prevent="handleGenerateCA">
        <div class="field">
          <label>证书名称</label>
          <input v-model="caForm.name" placeholder="例如：Nebula Root" required />
        </div>
        <div class="field">
          <label>描述信息</label>
          <input v-model="caForm.description" placeholder="用于标记证书用途" />
        </div>
        <div class="field">
          <label>有效期（天）</label>
          <input type="number" v-model.number="caForm.validity_days" min="1" />
        </div>
        <div class="actions">
          <button class="btn" type="submit">生成 / 替换根证书</button>
          <button class="btn secondary" type="button" :disabled="!ca" @click="downloadCA">下载根证书</button>
        </div>
      </form>
    </section>

    <section class="card">
      <h2>网络参数配置</h2>
      <p class="muted">这些设置会影响后续生成的节点配置文件。</p>
      <form class="form" @submit.prevent="saveSettings">
        <div class="field">
          <label>默认子网</label>
          <input v-model="settingsForm.default_subnet" placeholder="例如：10.10.0.0/24" />
        </div>
        <div class="field">
          <label>握手端口</label>
          <input type="number" v-model.number="settingsForm.handshake_port" min="1" max="65535" />
        </div>
        <div class="field">
          <label>证书有效期（天）</label>
          <input type="number" v-model.number="settingsForm.certificate_validity" min="1" />
        </div>
        <div class="field">
          <label>灯塔主机列表</label>
          <textarea
            v-model="settingsForm.lighthouse_hosts"
            rows="2"
            placeholder="多个主机请用逗号分隔，如：lighthouse1.example.com,lighthouse2.example.com"
          ></textarea>
        </div>
        <div class="field">
          <label>说明备注</label>
          <textarea v-model="settingsForm.description" rows="2" placeholder="可选"></textarea>
        </div>
        <button class="btn" type="submit">保存设置</button>
      </form>
    </section>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue';
import { downloadCACert, generateCA, getCA, getSettings, updateSettings } from '../api';

const ca = ref(null);
const caForm = reactive({
  name: 'Nebula Root',
  description: 'Nebula 管理平台自动签发',
  validity_days: 365
});

const settingsForm = reactive({
  default_subnet: '',
  handshake_port: 4242,
  certificate_validity: 365,
  lighthouse_hosts: '',
  description: ''
});

async function fetchCA() {
  try {
    const { data } = await getCA();
    ca.value = data.data;
  } catch (err) {
    console.error(err);
  }
}

function normaliseSettings(payload = {}) {
  return {
    default_subnet: payload.default_subnet ?? payload.DefaultSubnet ?? settingsForm.default_subnet,
    handshake_port: payload.handshake_port ?? payload.HandshakePort ?? settingsForm.handshake_port,
    certificate_validity:
      payload.certificate_validity ?? payload.CertificateValidity ?? settingsForm.certificate_validity,
    lighthouse_hosts: payload.lighthouse_hosts ?? payload.LighthouseHosts ?? settingsForm.lighthouse_hosts,
    description: payload.description ?? payload.Description ?? settingsForm.description
  };
}

async function fetchSettings() {
  try {
    const { data } = await getSettings();
    const normalised = normaliseSettings(data.data || {});
    Object.assign(settingsForm, normalised);
  } catch (err) {
    console.error(err);
  }
}

async function handleGenerateCA() {
  try {
    const payload = { ...caForm };
    const { data } = await generateCA(payload);
    ca.value = data.data;
    window.alert('根证书生成成功');
  } catch (err) {
    window.alert(err.response?.data?.error || '根证书生成失败');
  }
}

async function downloadCA() {
  try {
    const response = await downloadCACert();
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const link = document.createElement('a');
    link.href = url;
    link.download = 'nebula-ca.crt';
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (err) {
    window.alert('根证书下载失败，请稍后再试');
  }
}

async function saveSettings() {
  try {
    const payload = JSON.parse(JSON.stringify(settingsForm));
    const { data } = await updateSettings(payload);
    const normalised = normaliseSettings(data.data || {});
    Object.assign(settingsForm, normalised);
    window.alert('网络参数已保存');
  } catch (err) {
    window.alert(err.response?.data?.error || '保存失败');
  }
}

function formatDate(value) {
  if (!value) return '';
  return new Date(value).toLocaleString();
}

onMounted(() => {
  fetchCA();
  fetchSettings();
});
</script>

<style scoped>
.dashboard {
  display: grid;
  gap: 1.4rem;
}

.field {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.8rem;
}

.field label {
  font-weight: 600;
  margin-bottom: 0.3rem;
}

.actions {
  display: flex;
  gap: 0.6rem;
}

.muted {
  color: #64748b;
  font-size: 0.9rem;
}
</style>
