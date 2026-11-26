<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'
import Editor from '../components/Editor.vue'

const items = ref([])
const error = ref('')
const topicId = ref('')
const content = ref('')
const form = reactive({ topic_id: '' })

async function load() {
  if (!topicId.value) {
    return
  }
  error.value = ''
  try {
    const data = await api(`/admin/theory?topic_id=${topicId.value}`)
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/theory', {
      method: 'POST',
      body: JSON.stringify({ topic_id: topicId.value, content_md: content.value })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/theory/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(() => {
  form.topic_id = topicId.value
})
</script>

<template>
  <h2>Теория</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <input v-model="topicId" placeholder="ID темы" required />
    <button type="submit">Загрузить версии</button>
  </form>
  <Editor v-model="content" />
  <button :disabled="!topicId" @click="create">Создать версию</button>
  <table>
    <thead><tr><th>Версия</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.version }}</td>
        <td>{{ item.status }}</td>
        <td><button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button></td>
      </tr>
    </tbody>
  </table>
</template>
