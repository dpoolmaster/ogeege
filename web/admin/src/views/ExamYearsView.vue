<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const exams = ref([])
const error = ref('')
const form = reactive({ exam_id: '', year: new Date().getFullYear(), duration_minutes: 235 })

const examTitles = computed(() => {
  const map = {}
  for (const exam of exams.value) map[exam.id] = exam.title
  return map
})

async function load() {
  try {
    const data = await api('/admin/exam-years?limit=200')
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
    await api('/admin/exam-years', { method: 'POST', body: JSON.stringify(form) })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/exam-years/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(item) {
  error.value = ''
  try {
    await api(`/admin/exam-years/${item.id}`, { method: 'DELETE' })
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
  <h2>Учебные годы</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <select v-model="form.exam_id" required>
      <option value="" disabled>Выберите трек</option>
      <option v-for="exam in exams" :key="exam.id" :value="exam.id">{{ exam.title }}</option>
    </select>
    <input v-model.number="form.year" type="number" placeholder="Год" required />
    <input v-model.number="form.duration_minutes" type="number" placeholder="Длительность, мин" required />
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Трек</th><th>Год</th><th>Минуты</th><th>Статус</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ examTitles[item.exam_id] || item.exam_id }}</td>
        <td>{{ item.year }}</td>
        <td>{{ item.duration_minutes }}</td>
        <td>{{ item.status }}</td>
        <td class="admin-actions">
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
          <button class="secondary" @click="remove(item)">Удалить</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
