<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const form = reactive({ code: '', title: '', subject: '' })

async function load() {
  try {
    const data = await api('/admin/exams?limit=200')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/exams', { method: 'POST', body: JSON.stringify(form) })
    form.code = ''
    form.title = ''
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/exams/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(item) {
  error.value = ''
  try {
    await api(`/admin/exams/${item.id}`, { method: 'DELETE' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Треки</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <input v-model="form.code" placeholder="Код" required />
    <input v-model="form.title" placeholder="Название" required />
    <input v-model="form.subject" placeholder="Предмет" required />
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Код</th><th>Название</th><th>Предмет</th><th>Статус</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.code }}</td>
        <td>{{ item.title }}</td>
        <td>{{ item.subject }}</td>
        <td>{{ item.status }}</td>
        <td class="admin-actions">
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
          <button class="secondary" @click="remove(item)">Удалить</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
