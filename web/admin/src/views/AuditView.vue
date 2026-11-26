<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const filter = reactive({ actor_id: '', action: '', entity: '', from: '', to: '' })

async function load() {
  error.value = ''
  const params = new URLSearchParams({ limit: '100' })
  for (const key of Object.keys(filter)) {
    if (filter[key]) {
      params.set(key, filter[key])
    }
  }
  try {
    const data = await api('/admin/audit?' + params.toString())
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Аудит</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <input v-model="filter.actor_id" placeholder="ID актора" />
    <input v-model="filter.action" placeholder="Действие" />
    <input v-model="filter.entity" placeholder="Объект" />
    <input v-model="filter.from" placeholder="С RFC3339" />
    <input v-model="filter.to" placeholder="По RFC3339" />
    <button type="submit">Фильтр</button>
  </form>
  <table>
    <thead><tr><th>Время</th><th>Актор</th><th>Действие</th><th>Объект</th><th>IP</th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.created_at }}</td>
        <td>{{ item.actor_id }}</td>
        <td>{{ item.action }}</td>
        <td>{{ item.entity }} {{ item.entity_id }}</td>
        <td>{{ item.ip }}</td>
      </tr>
    </tbody>
  </table>
</template>
