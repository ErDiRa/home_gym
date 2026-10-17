import { watch } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { restoreSession } from './api'
import { isAuthenticated, state } from './session'

// Typed route meta. Children inherit their parent's meta (vue-router merges it into to.meta).
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guestOnly?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('./layouts/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [{ path: '', name: 'home', component: () => import('./views/HomeView.vue') }],
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('./views/LoginView.vue'),
      meta: { guestOnly: true },
    },
    { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
  ],
})

// Every navigation waits for the session restore, then redirects by the route's meta.
router.beforeEach(async (to) => {
  await restoreSession()
  const authenticated = isAuthenticated()
  if (to.meta.requiresAuth && !authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && authenticated) {
    return { name: 'home' }
  }
})

// Leave protected pages as soon as the session disappears (logout, expired or rejected cookie).
watch(
  () => state.user,
  (user) => {
    if (!user && router.currentRoute.value.meta.requiresAuth) {
      router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    }
  },
)

export default router
