<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'

const items = ref([])
const exams = ref([])
const error = ref('')
const form = reactive({ exam_id: '', parent_id: '', code: '', title: '', order_index: 1 })

const examTitles = computed(() => {
  const map = {}
  for (const exam of exams.value) map[exam.id] = exam.title
  return map
})

const topicTitles = computed(() => {
  const map = {}
  for (const topic of items.value) map[topic.id] = topic.title
  return map
})

const parentOptions = computed(() =>
  items.value.filter((topic) => topic.exam_id === form.exam_id),
)

watch(
  () => form.exam_id,
  () => {
    form.parent_id = ''
  },
)

async function load() {
  error.value = ''
  try {
    const data = await api('/admin/topics?limit=200')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function loadExams() {
  try {
    const data = await api('/admin/exams?limit=200')
    exams.value = data.items || []
    if (!form.exam_id && exams.value.length > 0) form.exam_id = exams.value[0].id
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

onMounted(() => {
  load()
  loadExams()
})
</script>

<template>
  <h2>Темы</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <select v-model="form.exam_id" required>
      <option value="" disabled>Выберите трек</option>
      <option v-for="exam in exams" :key="exam.id" :value="exam.id">{{ exam.title }}</option>
    </select>
    <select v-model="form.parent_id">
      <option value="">Без родителя</option>
      <option v-for="topic in parentOptions" :key="topic.id" :value="topic.id">{{ topic.title }}</option>
    </select>
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
        <td>{{ examTitles[item.exam_id] || item.exam_id }}</td>
        <td>{{ item.parent_id ? topicTitles[item.parent_id] || item.parent_id : '' }}</td>
        <td><button class="secondary" @click="remove(item)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>
