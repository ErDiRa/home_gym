import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { BoltIcon, ClipboardDocumentListIcon, DevicePhoneMobileIcon } from '@heroicons/vue/24/outline'

export interface NavItem {
  name: string
  to: RouteLocationRaw
  icon: Component
}

// Entries of the dashboard's side navigation; each one is a child route of the dashboard
// layout, except the tracker, which has its own mobile layout.
export const navigation: NavItem[] = [
  { name: 'Exercises', to: { name: 'exercises' }, icon: BoltIcon },
  { name: 'Trainings', to: { name: 'trainings' }, icon: ClipboardDocumentListIcon },
  { name: 'Tracker', to: { name: 'track' }, icon: DevicePhoneMobileIcon },
]
