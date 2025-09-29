<template>
  <div class="templates">
    <section class="card">
      <h2>配置模版管理</h2>
      <div class="layout">
        <aside>
          <ul>
            <li
              v-for="tpl in templates"
              :key="tpl.id"
              :class="{ active: tpl.name === currentName }"
              @click="selectTemplate(tpl)"
            >
              {{ tpl.name }}
            </li>
          </ul>
          <button class="btn" type="button" @click="newTemplate">新建模版</button>
        </aside>
        <div class="editor" v-if="current">
          <div class="field">
            <label>模版名称</label>
            <input v-model="current.name" placeholder="例如：default" />
          </div>
          <div class="field">
            <label>模版内容</label>
            <textarea v-model="current.content" rows="18"></textarea>
          </div>
          <div class="actions">
            <button class="btn" type="button" @click="save">保存</button>
            <button class="btn secondary" type="button" @click="remove" :disabled="!current.id">删除</button>
          </div>
        </div>
        <div v-else class="empty">请选择左侧模版或创建新的模版。</div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue';
import { deleteTemplate, listTemplates, upsertTemplate } from '../api';

const templates = ref([]);
const current = reactive({ id: null, name: '', content: '' });

const currentName = computed(() => current.name);

function selectTemplate(tpl) {
  Object.assign(current, tpl);
}

function newTemplate() {
  Object.assign(current, { id: null, name: '', content: '' });
}

async function load() {
  try {
    const { data } = await listTemplates();
    templates.value = data.data || [];
    if (templates.value.length) {
      selectTemplate(templates.value[0]);
    } else {
      newTemplate();
    }
  } catch (err) {
    console.error(err);
  }
}

async function save() {
  if (!current.name) {
    window.alert('模版名称不能为空');
    return;
  }
  try {
    const payload = { name: current.name, content: current.content };
    await upsertTemplate(payload);
    window.alert('模版已保存');
    await load();
  } catch (err) {
    window.alert(err.response?.data?.error || '保存失败');
  }
}

async function remove() {
  if (!current.id) {
    window.alert('请选择一个已经存在的模版');
    return;
  }
  if (!window.confirm(`确认删除模版 ${current.name} 吗？`)) {
    return;
  }
  try {
    await deleteTemplate(current.id);
    window.alert('模版已删除');
    newTemplate();
    await load();
  } catch (err) {
    window.alert(err.response?.data?.error || '删除失败');
  }
}

onMounted(load);
</script>

<style scoped>
.layout {
  display: flex;
  gap: 1.4rem;
}

aside {
  width: 180px;
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
}

aside ul {
  list-style: none;
  margin: 0;
  padding: 0;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
}

aside li {
  padding: 0.6rem 0.8rem;
  cursor: pointer;
  border-bottom: 1px solid #e2e8f0;
}

aside li:last-child {
  border-bottom: none;
}

aside li.active {
  background-color: #e0f2fe;
  font-weight: 600;
}

.editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.field textarea {
  min-height: 320px;
  font-family: 'SFMono-Regular', Consolas, monospace;
}

.actions {
  display: flex;
  gap: 0.6rem;
}

.empty {
  padding: 2rem;
  color: #64748b;
}
</style>
