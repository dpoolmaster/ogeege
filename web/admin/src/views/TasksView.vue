<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'
import Editor from '../components/Editor.vue'

const items = ref([])
const error = ref('')
const editingId = ref('')
const importText = ref('')
const uploadFile = ref(null)
const uploadTaskId = ref('')
const filter = reactive({ exam_year_id: '', kim_number_id: '', status: '', query: '' })
const form = reactive({
  exam_year_id: '',
  kim_number_id: '',
  topic_id: '',
  difficulty: 'easy',
  statement_md: '',
  answer_spec: '{"kind":"number","accepted":["1"]}',
  answer_display: '1',
  source: 'seed',
  source_ref: ''
})

async function load() {
  error.value = ''
  const params = new URLSearchParams({ limit: '100' })
  for (const key of Object.keys(filter)) {
    if (filter[key]) {
      params.set(key, filter[key])
    }
  }
  try {
    const data = await api('/admin/tasks?' + params.toString())
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

function edit(item) {
  editingId.value = item.id
  form.exam_year_id = item.exam_year_id
  form.kim_number_id = item.kim_number_id
  form.topic_id = item.topic_id || ''
  form.difficulty = item.difficulty
  form.statement_md = item.statement_md
  form.answer_spec = JSON.stringify(item.answer_spec)
  form.answer_display = item.answer_display
  form.source = item.source
  form.source_ref = item.source_ref
}

function resetForm() {
  editingId.value = ''
  form.statement_md = ''
  form.answer_spec = '{"kind":"number","accepted":["1"]}'
  form.answer_display = '1'
}

async function save() {
  error.value = ''
  const payload = { ...form, topic_id: form.topic_id || null, answer_spec: JSON.parse(form.answer_spec) }
  try {
    if (editingId.value) {
      await api(`/admin/tasks/${editingId.value}`, { method: 'PATCH', body: JSON.stringify(payload) })
    } else {
      await api('/admin/tasks', { method: 'POST', body: JSON.stringify(payload) })
    }
    resetForm()
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  await act(`/admin/tasks/${item.id}/publish`, 'POST')
}

async function archive(item) {
  await act(`/admin/tasks/${item.id}/archive`, 'POST')
}

async function remove(item) {
  await act(`/admin/tasks/${item.id}`, 'DELETE')
}

async function act(path, method) {
  error.value = ''
  try {
    await api(path, { method })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function importTasks() {
  error.value = ''
  try {
    const items = JSON.parse(importText.value)
    const report = await api('/admin/tasks/import', { method: 'POST', body: JSON.stringify({ items }) })
    error.value = `Создано: ${report.created.length}, пропущено: ${report.skipped.length}, отклонено: ${report.rejected.length}`
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function upload() {
  if (!uploadTaskId.value || !uploadFile.value) {
    return
  }
  error.value = ''
  const data = new FormData()
  data.append('file', uploadFile.value)
  try {
    await api(`/admin/tasks/${uploadTaskId.value}/attachments`, { method: 'POST', body: data })
    uploadFile.value = null
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Задания</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>

  <form class="admin-form" @submit.prevent="load">
    <input v-model="filter.exam_year_id" placeholder="ID учебного года" />
    <input v-model="filter.kim_number_id" placeholder="ID номера КИМ" />
    <select v-model="filter.status">
      <option value="">Любой статус</option>
      <option value="draft">Черновик</option>
      <option value="published">Опубликовано</option>
      <option value="archived">Архив</option>
    </select>
    <input v-model="filter.query" placeholder="Поиск" />
    <button type="submit">Фильтр</button>
  </form>

  <h3>{{ editingId ? 'Изменить задание' : 'Новое задание' }}</h3>
  <form class="admin-form" @submit.prevent="save">
    <input v-model="form.exam_year_id" placeholder="ID учебного года" required />
    <input v-model="form.kim_number_id" placeholder="ID номера КИМ" required />
    <input v-model="form.topic_id" placeholder="ID темы" />
    <select v-model="form.difficulty">
      <option value="easy">Лёгкое</option>
      <option value="medium">Среднее</option>
      <option value="hard">Сложное</option>
    </select>
    <input v-model="form.source" placeholder="Источник" required />
    <input v-model="form.source_ref" placeholder="Ссылка на источник" />
    <input v-model="form.answer_display" placeholder="Отображаемый ответ" />
    <textarea v-model="form.answer_spec" placeholder="answer_spec JSON"></textarea>
    <button type="submit">Сохранить</button>
    <button v-if="editingId" type="button" class="secondary" @click="resetForm">Отмена</button>
  </form>
  <Editor v-model="form.statement_md" />

  <h3>Импорт</h3>
  <textarea v-model="importText" placeholder='[{"exam_year_id":"...","kim_number_id":"...","difficulty":"easy","statement_md":"...","answer_spec":{"kind":"number","accepted":["1"]},"source":"seed"}]'></textarea>
  <button @click="importTasks">Импортировать</button>

  <h3>Изображение задания</h3>
  <form class="admin-form" @submit.prevent="upload">
    <input v-model="uploadTaskId" placeholder="ID задания" required />
    <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" @change="uploadFile = $event.target.files[0]" />
    <button type="submit">Загрузить</button>
  </form>

  <table>
    <thead><tr><th>Условие</th><th>Статус</th><th>Версия</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.statement_md.slice(0, 80) }}</td>
        <td>{{ item.status }}</td>
        <td>{{ item.version }}</td>
        <td class="admin-actions">
          <button @click="edit(item)">Изменить</button>
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
          <button v-if="item.status === 'published'" class="secondary" @click="archive(item)">В архив</button>
          <button class="secondary" @click="remove(item)">Удалить</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
