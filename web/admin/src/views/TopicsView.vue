<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'

const allItems = ref([])
const exams = ref([])
const error = ref('')
const filter = reactive({ exam_id: '', parent_id: '', code: '' })
const form = reactive({ exam_id: '', parent_id: '', code: '', title: '', order_index: 1 })

const examTitles = computed(() => {
  const map = {}
  for (const exam of exams.value) map[exam.id] = exam.title
  return map
})

const filteredItems = computed(() => {
  let list = allItems.value
  if (filter.exam_id) list = list.filter((topic) => topic.exam_id === filter.exam_id)
  const code = filter.code.trim().toLowerCase()
  if (code) list = list.filter((topic) => topic.code.toLowerCase().includes(code))
  if (filter.parent_id) {
    const byId = new Map(list.map((topic) => [topic.id, topic]))
    if (!byId.has(filter.parent_id)) return []
    const children = new Map()
    for (const topic of list) {
      if (!topic.parent_id) continue
      const siblings = children.get(topic.parent_id) || []
      siblings.push(topic)
      children.set(topic.parent_id, siblings)
    }
    const subtree = new Set()
    const visit = (id) => {
      if (subtree.has(id)) return
      subtree.add(id)
      for (const child of children.get(id) || []) visit(child.id)
    }
    visit(filter.parent_id)
    list = list.filter((topic) => subtree.has(topic.id))
  }
  return list
})

const rows = computed(() => {
  const byId = new Map(filteredItems.value.map((topic) => [topic.id, topic]))
  const children = new Map()
  const roots = []
  for (const topic of filteredItems.value) {
    if (topic.parent_id && byId.has(topic.parent_id)) {
      const list = children.get(topic.parent_id) || []
      list.push(topic)
      children.set(topic.parent_id, list)
    } else {
      roots.push(topic)
    }
  }

  const sortTopics = (a, b) => a.order_index - b.order_index || a.title.localeCompare(b.title)
  const result = []
  const visited = new Set()
  const visit = (topic, depth) => {
    if (visited.has(topic.id)) return
    visited.add(topic.id)
    result.push({ ...topic, depth })
    for (const child of (children.get(topic.id) || []).slice().sort(sortTopics)) {
      visit(child, depth + 1)
    }
  }
  for (const root of roots.slice().sort(sortTopics)) visit(root, 0)

  return result
})

const formParentOptions = computed(() =>
  allItems.value.filter((topic) => topic.exam_id === form.exam_id),
)

const filterParentOptions = computed(() =>
  filter.exam_id ? allItems.value.filter((topic) => topic.exam_id === filter.exam_id) : [],
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
    const data = await api('/admin/topics?limit=200')
    allItems.value = data.items || []
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
  if (!window.confirm(`Удалить тему «${item.title}»?`)) return
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
      <select v-model="filter.parent_id" :disabled="!filter.exam_id">
        <option value="">{{ filter.exam_id ? 'Любая родительская тема' : 'Сначала выберите трек' }}</option>
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
      <tr><th>Порядок</th><th>Код</th><th>Название</th><th>Трек</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="row in rows" :key="row.id">
        <td>{{ row.order_index }}</td>
        <td>{{ row.code }}</td>
        <td :style="{ paddingLeft: row.depth * 1.5 + 'rem' }">
          <span v-if="row.depth > 0">└ </span>{{ row.title }}
        </td>
        <td>{{ examTitles[row.exam_id] || row.exam_id }}</td>
        <td><button class="secondary" @click="remove(row)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>
