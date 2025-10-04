<template>
  <div class="login-wrapper">
    <div class="login-card">
      <h1 class="title">Nebula 管理控制台</h1>
      <form class="login-form" @submit.prevent="onSubmit">
        <label class="field">
          <span>用户名</span>
          <input
            v-model.trim="form.username"
            type="text"
            autocomplete="username"
            required
            placeholder="请输入用户名"
            @input="clearError"
          />
        </label>
        <label class="field">
          <span>密码</span>
          <input
            v-model.trim="form.password"
            type="password"
            autocomplete="current-password"
            required
            placeholder="请输入密码"
            @input="clearError"
          />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button type="submit" :disabled="submitting">
          {{ submitting ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuth } from '../composables/useAuth';

const router = useRouter();
const route = useRoute();
const { state, login, clearError } = useAuth();

const form = reactive({
  username: '',
  password: ''
});

const submitting = computed(() => state.loading);
const error = computed(() => state.error);

const onSubmit = async () => {
  const ok = await login(form.username, form.password);
  if (ok) {
    const redirectTarget = route.query.redirect || '/dashboard';
    router.replace(redirectTarget);
  }
};
</script>

<style scoped>
.login-wrapper {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at top, rgba(148, 163, 184, 0.35), rgba(15, 23, 42, 0.4)), #e2e8f0;
  backdrop-filter: blur(24px);
}

.login-card {
  background: rgba(255, 255, 255, 0.55);
  border-radius: 18px;
  padding: 2.6rem 2.9rem;
  width: min(420px, 92%);
  box-shadow: 0 25px 55px rgba(15, 23, 42, 0.22);
  backdrop-filter: blur(20px) saturate(160%);
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.title {
  margin-bottom: 1.8rem;
  text-align: center;
  color: #0f172a;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.2rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.field span {
  font-size: 0.9rem;
  color: #334155;
}

.field input {
  padding: 0.65rem 0.75rem;
  border-radius: 6px;
  border: 1px solid rgba(148, 163, 184, 0.6);
  font-size: 0.95rem;
  transition: border-color 0.2s ease;
}

.field input:focus {
  border-color: #38bdf8;
  outline: none;
  box-shadow: 0 0 0 2px rgba(56, 189, 248, 0.2);
}

button[type='submit'] {
  margin-top: 0.5rem;
  padding: 0.65rem;
  border: none;
  border-radius: 6px;
  background: linear-gradient(120deg, #0ea5e9, #2563eb);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

button[disabled] {
  opacity: 0.7;
  cursor: not-allowed;
}

.error {
  color: #dc2626;
  font-size: 0.85rem;
  margin-top: -0.5rem;
}
</style>
