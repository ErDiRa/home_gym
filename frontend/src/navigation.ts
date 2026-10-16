import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { BoltIcon, ClipboardDocumentListIcon, UsersIcon } from '@heroicons/vue/24/outline'

export interface NavItem {
  name: string
  to: RouteLocationRaw
  icon: Component
  // Only shown to admins (the routes also require it).
  adminOnly?: boolean
}

// Entries of the side navigation; each one is a child route of the app layout.
export const navigation: NavItem[] = [
  { name: 'Exercises', to: { name: 'exercises' }, icon: BoltIcon },
  { name: 'Trainings', to: { name: 'trainings' }, icon: ClipboardDocumentListIcon, adminOnly: true },
  { name: 'Clients', to: { name: 'clients' }, icon: UsersIcon, adminOnly: true },
]
