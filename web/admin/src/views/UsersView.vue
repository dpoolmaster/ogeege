<script setup>
import { onMounted, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const query = ref('')
const error = ref('')
const roles = ref({})
const statuses = ref({})

async function load() {
  error.value = ''
  try {
    const data = await api('/admin/users?limit=100&query=' + encodeURIComponent(query.value))
    items.value = data.items || []
    for (const item of items.value) {
      roles.value[item.id] = item.role
      statuses.value[item.id] = item.status
    }
  } catch (err) {
    error.value = err.message
  }
}

async function save(item) {
  error.value = ''
  try {
    await api(`/admin/users/${item.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ role: roles.value[item.id], status: statuses.value[item.id] })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Пользователи</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <input v-model="query" placeholder="Email или ID" />
    <button type="submit">Найти</button>
  </form>
  <table>
    <thead><tr><th>Email</th><th>Роль</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.email }}</td>
        <td>
          <select v-model="roles[item.id]">
            <option value="student">student</option>
            <option value="parent">parent</option>
            <option value="methodist">methodist</option>
            <option value="admin">admin</option>
          </select>
        </td>
        <td>
          <select v-model="statuses[item.id]">
            <option value="unverified">unverified</option>
            <option value="active">active</option>
            <option value="blocked">blocked</option>
            <option value="anonymized">anonymized</option>
          </select>
        </td>
        <td><button @click="save(item)">Сохранить</button></td>
      </tr>
    </tbody>
  </table>
</template>
