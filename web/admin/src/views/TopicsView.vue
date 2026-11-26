<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'

const items = ref([])
const exams = ref([])
const error = ref('')
const filter = reactive({ exam_id: '', parent_id: '', code: '' })
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

const formParentOptions = computed(() =>
  items.value.filter((topic) => topic.exam_id === form.exam_id),
)

const filterParentOptions = computed(() =>
  filter.exam_id ? items.value.filter((topic) => topic.exam_id === filter.exam_id) : items.value,
)

watch(
  () => form.exam_id,
  () => {
    form.parent_id = ''
  },
)

watch(
  () => filter.exam_id,
  () => {
    filter.parent_id = ''
  },
)

async function load() {
  error.value = ''
  try {
    const params = new URLSearchParams({ limit: '200' })
    if (filter.exam_id) params.set('exam_id', filter.exam_id)
    if (filter.parent_id) params.set('parent_id', filter.parent_id)
    if (filter.code.trim()) params.set('code', filter.code.trim())
    const data = await api('/admin/topics?' + params.toString())
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
  <form class="admin-form" @submit.prevent="load">
    <label>
      Трек
      <select v-model="filter.exam_id">
        <option value="">Все треки</option>
        <option v-for="exam in exams" :key="exam.id" :value="exam.id">{{ exam.title }}</option>
      </select>
    </label>
    <label>
      Родительская тема
      <select v-model="filter.parent_id">
        <option value="">Любой родитель</option>
        <option v-for="topic in filterParentOptions" :key="topic.id" :value="topic.id">{{ topic.title }}</option>
      </select>
    </label>
    <label>
      Код
      <input v-model="filter.code" placeholder="Часть кода" />
    </label>
    <button type="submit">Фильтр</button>
  </form>
  <form class="admin-form" @submit.prevent="create">
    <label>
      Трек
      <select v-model="form.exam_id" required>
        <option value="" disabled>Выберите трек</option>
        <option v-for="exam in exams" :key="exam.id" :value="exam.id">{{ exam.title }}</option>
      </select>
    </label>
    <label>
      Родительская тема
      <select v-model="form.parent_id">
        <option value="">Без родителя</option>
        <option v-for="topic in formParentOptions" :key="topic.id" :value="topic.id">{{ topic.title }}</option>
      </select>
    </label>
    <label>
      Код
      <input v-model="form.code" placeholder="Код" required />
    </label>
    <label>
      Название
      <input v-model="form.title" placeholder="Название" required />
    </label>
    <label>
      Порядок
      <input v-model.number="form.order_index" type="number" placeholder="Порядок" />
    </label>
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Порядок</th><th>Код</th><th>Название</th><th>Трек</th><th>Родительская тема</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.order_index }}</td>
        <td>{{ item.code }}</td>
        <td>{{ item.title }}</td>
        <td>{{ examTitles[item.exam_id] || item.exam_id }}</td>
        <td>{{ item.parent_id ? topicTitles[item.parent_id] || item.parent_id : '' }}</td>
        <td><button class="secondary" @click="remove(item)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>
