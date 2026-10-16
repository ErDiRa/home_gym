import { watch } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { restoreSession } from './api'
import { isAuthenticated, state } from './session'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
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
      children: [
        { path: '', name: 'home', redirect: { name: 'exercises' } },
        { path: 'exercises', name: 'exercises', component: () => import('./views/exercises/ExercisesView.vue') },
        {
          path: 'exercises/new',
          name: 'exercise-new',
          component: () => import('./views/exercises/ExerciseFormView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          // Admins edit the exercise; user sessions see the same form read-only.
          path: 'exercises/:id',
          name: 'exercise-edit',
          component: () => import('./views/exercises/ExerciseFormView.vue'),
        },
        {
          path: 'trainings',
          name: 'trainings',
          component: () => import('./views/trainings/TrainingsView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          // ?client=<id> preselects the client.
          path: 'trainings/new',
          name: 'training-new',
          component: () => import('./views/trainings/TrainingFormView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'trainings/:id',
          name: 'training-edit',
          component: () => import('./views/trainings/TrainingFormView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'clients',
          name: 'clients',
          component: () => import('./views/clients/ClientsView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'clients/new',
          name: 'client-new',
          component: () => import('./views/clients/ClientFormView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'clients/:id',
          name: 'client-edit',
          component: () => import('./views/clients/ClientFormView.vue'),
          meta: { requiresAdmin: true },
        },
      ],
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('./views/login/LoginView.vue'),
      meta: { guestOnly: true },
    },
    { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
  ],
})

router.beforeEach(async (to) => {
  await restoreSession()
  const authenticated = isAuthenticated()
  if (to.meta.requiresAuth && !authenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin && !state.user?.is_admin) {
    return { name: 'exercises' }
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
