<template>
  <div class="layout">
    <NavBar v-if="isAuthenticated" />
    <main class="content">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue';
import NavBar from './components/NavBar.vue';
import { useAuth } from './composables/useAuth';

const { state, ensureSession } = useAuth();

onMounted(() => {
  ensureSession();
});

const isAuthenticated = computed(() => Boolean(state.user));
</script>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.content {
  flex: 1;
  padding: 1.5rem;
  max-width: 1100px;
  margin: 0 auto;
}
</style>
