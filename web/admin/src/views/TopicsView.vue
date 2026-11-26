<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const form = reactive({ exam_id: '', parent_id: '', code: '', title: '', order_index: 1 })

async function load() {
  error.value = ''
  try {
    const data = await api('/admin/topics?limit=200')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    const payload = { ...form, parent_id: form.parent_id || null }
    await api('/admin/topics', { method: 'POST', body: JSON.stringify(payload) })
    form.code = ''
    form.title = ''
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(item) {
  error.value = ''
  try {
    await api(`/admin/topics/${item.id}`, { method: 'DELETE' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Темы</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <input v-model="form.exam_id" placeholder="ID трека" required />
    <input v-model="form.parent_id" placeholder="ID родителя (необязательно)" />
    <input v-model="form.code" placeholder="Код" required />
    <input v-model="form.title" placeholder="Название" required />
    <input v-model.number="form.order_index" type="number" placeholder="Порядок" />
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Код</th><th>Название</th><th>Трек</th><th>Родитель</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.code }}</td>
        <td>{{ item.title }}</td>
        <td>{{ item.exam_id }}</td>
        <td>{{ item.parent_id }}</td>
        <td><button class="secondary" @click="remove(item)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>
