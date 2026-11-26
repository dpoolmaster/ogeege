<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import Editor from '../components/Editor.vue'

const route = useRoute()
const router = useRouter()

const items = ref([])
const error = ref('')
const topicId = ref('')
const content = ref('')
const selectedId = ref('')
const exams = ref([])
const topics = ref([])

const selected = computed(() => items.value.find((item) => item.id === selectedId.value) || null)

const groups = computed(() => {
  const byExam = new Map()
  for (const topic of topics.value) {
    if (!byExam.has(topic.exam_id)) {
      byExam.set(topic.exam_id, [])
    }
    byExam.get(topic.exam_id).push(topic)
  }

  return exams.value
    .map((exam) => ({
      id: exam.id,
      title: exam.title,
      options: flatten(byExam.get(exam.id) || [])
    }))
    .filter((group) => group.options.length > 0)
})

function flatten(list) {
  const sorted = [...list].sort((a, b) => a.order_index - b.order_index || a.title.localeCompare(b.title))
  const children = new Map()
  const roots = []
  for (const topic of sorted) {
    if (topic.parent_id) {
      if (!children.has(topic.parent_id)) {
        children.set(topic.parent_id, [])
      }
      children.get(topic.parent_id).push(topic)
    } else {
      roots.push(topic)
    }
  }

  const result = []
  const visit = (topic, depth) => {
    result.push({ id: topic.id, label: `${'— '.repeat(depth)}${topic.title}` })
    for (const child of children.get(topic.id) || []) {
      visit(child, depth + 1)
    }
  }
  for (const root of roots) {
    visit(root, 0)
  }

  return result
}

async function loadCatalog() {
  error.value = ''
  try {
    const examData = await api('/admin/exams?limit=200')
    exams.value = examData.items || []

    const all = []
    let cursor = ''
    for (;;) {
      const params = new URLSearchParams({ limit: '200' })
      if (cursor) {
        params.set('cursor', cursor)
      }
      const data = await api('/admin/topics?' + params.toString())
      all.push(...(data.items || []))
      if (!data.next_cursor) {
        break
      }
      cursor = data.next_cursor
    }
    topics.value = all
  } catch (err) {
    error.value = err.message
  }
}

async function loadVersions() {
  const data = await api(`/admin/theory?topic_id=${topicId.value}`)
  items.value = data.items || []
}

async function loadActive() {
  try {
    const active = await api(`/admin/theory/active?topic_id=${topicId.value}`)
    content.value = active.content_md || ''
    selectedId.value = active.id || ''
  } catch (err) {
    if (err.status !== 404) {
      throw err
    }
    content.value = ''
    selectedId.value = ''
  }
}

async function load() {
  if (!topicId.value) {
    return
  }
  error.value = ''
  try {
    await loadVersions()
    await loadActive()
  } catch (err) {
    error.value = err.message
  }
}

function changeTopic() {
  if (!topicId.value) {
    return
  }
  router.replace({ query: { ...route.query, topic_id: topicId.value } })
  load()
}

function selectVersion(item) {
  content.value = item.content_md || ''
  selectedId.value = item.id
}

async function create() {
  error.value = ''
  try {
    const created = await api('/admin/theory', {
      method: 'POST',
      body: JSON.stringify({ topic_id: topicId.value, content_md: content.value })
    })
    selectedId.value = created.id
    await loadVersions()
  } catch (err) {
    error.value = err.message
  }
}

async function updateVersion() {
  if (!selected.value) {
    return
  }
  error.value = ''
  try {
    await api(`/admin/theory/${selected.value.id}`, {
      method: 'PATCH',
      body: JSON.stringify({ content_md: content.value })
    })
    await loadVersions()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/theory/${item.id}/publish`, { method: 'POST' })
    await loadVersions()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(async () => {
  await loadCatalog()
  const initial = route.query.topic_id
  if (typeof initial === 'string' && initial) {
    topicId.value = initial
    await load()
  }
})
</script>

<template>
  <h2>Теория</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="changeTopic">
    <select v-model="topicId" required @change="changeTopic">
      <option value="" disabled>Выберите тему</option>
      <optgroup v-for="group in groups" :key="group.id" :label="group.title">
        <option v-for="option in group.options" :key="option.id" :value="option.id">{{ option.label }}</option>
      </optgroup>
    </select>
    <button type="submit">Загрузить версии</button>
  </form>
  <Editor v-model="content" />
  <div class="admin-actions admin-actions-bar">
    <button :disabled="!topicId" @click="create">Создать версию</button>
    <button v-if="selected && selected.status === 'draft'" @click="updateVersion">Сохранить изменения</button>
  </div>
  <table>
    <thead><tr><th>Версия</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr
        v-for="item in items"
        :key="item.id"
        class="theory-row"
        :class="{ selected: item.id === selectedId }"
        @click="selectVersion(item)"
      >
        <td>{{ item.version }}</td>
        <td>{{ item.status }}</td>
        <td><button v-if="item.status !== 'published'" @click.stop="publish(item)">Опубликовать</button></td>
      </tr>
    </tbody>
  </table>
</template>
