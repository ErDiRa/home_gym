<script setup>
import { ref, computed } from 'vue'
import AccessoryCard from './components/AccessoryCard.vue'

const accessories = ref([
  { id: 1, name: 'Leather Wallet', category: 'Wallets', price: 59 },
  { id: 2, name: 'Silver Watch', category: 'Watches', price: 249 },
  { id: 3, name: 'Wool Scarf', category: 'Scarves', price: 39 },
  { id: 4, name: 'Aviator Sunglasses', category: 'Eyewear', price: 129 },
  { id: 5, name: 'Canvas Belt', category: 'Belts', price: 29 },
  { id: 6, name: 'Leather Gloves', category: 'Gloves', price: 49 },
])

const query = ref('')
const favorites = ref(new Set())

const filtered = computed(() =>
  accessories.value.filter((a) =>
    `${a.name} ${a.category}`.toLowerCase().includes(query.value.toLowerCase()),
  ),
)

function toggleFavorite(id) {
  const next = new Set(favorites.value)
  next.has(id) ? next.delete(id) : next.add(id)
  favorites.value = next
}
</script>

<template>
  <div class="min-h-screen bg-slate-50 text-slate-800">
    <header class="bg-white shadow-sm">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
        <h1 class="text-xl font-bold text-indigo-600">My Accessories</h1>
        <span class="text-sm text-slate-500">♥ {{ favorites.size }} favorites</span>
      </div>
    </header>

    <main class="mx-auto max-w-5xl px-4 py-8">
      <input
        v-model="query"
        type="search"
        placeholder="Search accessories..."
        class="mb-6 w-full rounded-lg border border-slate-300 bg-white px-4 py-2 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
      />

      <div v-if="filtered.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <AccessoryCard
          v-for="item in filtered"
          :key="item.id"
          :item="item"
          :favorite="favorites.has(item.id)"
          @toggle="toggleFavorite(item.id)"
        />
      </div>
      <p v-else class="text-center text-slate-500">No accessories found.</p>
    </main>
  </div>
</template>
