<template>
  <nav class="navbar">
    <div class="brand">Nebula 管理平台</div>
    <div class="links">
      <RouterLink to="/dashboard" active-class="active">总览</RouterLink>
      <RouterLink to="/nodes" active-class="active">节点管理</RouterLink>
      <RouterLink to="/templates" active-class="active">配置模板</RouterLink>
    </div>
    <div class="account" v-if="user">
      <span class="username">{{ user.username }}</span>
      <button class="logout" @click="handleLogout">退出登录</button>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';

const router = useRouter();
const { state, logout } = useAuth();

const user = computed(() => state.user);

const handleLogout = async () => {
  await logout();
  router.push({ name: 'login' });
};
</script>

<style scoped>
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #1e293b;
  color: #fff;
  padding: 0.9rem 1.6rem;
}

.brand {
  font-weight: 700;
  font-size: 1.1rem;
}

.links {
  display: flex;
  gap: 1rem;
}

.links a {
  color: #cbd5f5;
  font-weight: 600;
}

.links a.active {
  color: #fff;
  border-bottom: 2px solid #38bdf8;
}

.account {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.username {
  font-size: 0.95rem;
  color: #cbd5f5;
}

.logout {
  background: transparent;
  border: 1px solid rgba(255, 255, 255, 0.6);
  color: #fff;
  padding: 0.35rem 0.75rem;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.logout:hover {
  background: rgba(255, 255, 255, 0.1);
}
</style>
