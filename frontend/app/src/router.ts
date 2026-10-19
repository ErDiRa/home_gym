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

// The tracker is made for phones and the dashboard for larger screens; "/" picks by width.
const isPhone = () => window.matchMedia('(max-width: 767px)').matches

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', redirect: () => ({ name: isPhone() ? 'track' : 'dashboard' }) },
    {
      // Planning (desktop first): exercises, the session library and trainings; later analytics and the wiki.
      path: '/dashboard',
      component: () => import('./layouts/DashboardLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'dashboard', redirect: { name: 'trainings' } },
        { path: 'exercises', name: 'exercises', component: () => import('./views/exercises/ExercisesView.vue') },
        {
          path: 'exercises/new',
          name: 'exercise-new',
          component: () => import('./views/exercises/ExerciseFormView.vue'),
        },
        {
          path: 'exercises/:id',
          name: 'exercise-edit',
          component: () => import('./views/exercises/ExerciseFormView.vue'),
        },
        { path: 'sessions', name: 'sessions', component: () => import('./views/sessions/SessionsView.vue') },
        {
          path: 'sessions/new',
          name: 'session-new',
          component: () => import('./views/sessions/SessionFormView.vue'),
        },
        {
          path: 'sessions/:id',
          name: 'session-edit',
          component: () => import('./views/sessions/SessionFormView.vue'),
        },
        {
          path: 'trainings',
          name: 'trainings',
          component: () => import('./views/trainings/TrainingsView.vue'),
        },
        {
          path: 'trainings/new',
          name: 'training-new',
          component: () => import('./views/trainings/TrainingFormView.vue'),
        },
        {
          path: 'trainings/:id',
          name: 'training-edit',
          component: () => import('./views/trainings/TrainingFormView.vue'),
        },
      ],
    },
    {
      // Tracking (mobile first): this week's sessions, the workout in progress and the history.
      path: '/track',
      component: () => import('./layouts/TrackLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'track', component: () => import('./views/track/TrackHomeView.vue') },
        { path: 'workouts/:id', name: 'workout', component: () => import('./views/track/WorkoutView.vue') },
        { path: 'history', name: 'history', component: () => import('./views/track/HistoryView.vue') },
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
