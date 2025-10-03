import { createRouter, createWebHistory } from 'vue-router';
import DashboardView from '../views/DashboardView.vue';
import NodesView from '../views/NodesView.vue';
import TemplatesView from '../views/TemplatesView.vue';
import NodeNetworkView from '../views/NodeNetworkView.vue';
import LoginView from '../views/LoginView.vue';
import { useAuth } from '../composables/useAuth';

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', name: 'login', component: LoginView, meta: { requiresAuth: false } },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  { path: '/nodes', name: 'nodes', component: NodesView },
  { path: '/nodes/:id/network', name: 'node-network', component: NodeNetworkView },
  { path: '/templates', name: 'templates', component: TemplatesView }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach(async (to, from, next) => {
  const { state, ensureSession } = useAuth();
  if (!state.initialized) {
    await ensureSession();
  }

  if (to.meta.requiresAuth === false) {
    if (to.name === 'login' && state.user) {
      const redirectTarget = to.query.redirect || '/dashboard';
      next(redirectTarget);
      return;
    }
    next();
    return;
  }

  if (!state.user) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  next();
});

export default router;
