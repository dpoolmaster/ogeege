<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api'
import Editor from '../components/Editor.vue'

const items = ref([])
const examYears = ref([])
const kimNumbers = ref([])
const topics = ref([])
const exams = ref([])
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

const examTitles = computed(() => {
  const map = {}
  for (const exam of exams.value) map[exam.id] = exam.title
  return map
})

const examYearLabels = computed(() => {
  const map = {}
  for (const year of examYears.value) {
    const title = examTitles.value[year.exam_id]
    map[year.id] = title ? `${title} — ${year.year}` : String(year.year)
  }
  return map
})

const kimNumberLabels = computed(() => {
  const map = {}
  for (const number of kimNumbers.value) {
    map[number.id] = number.title
      ? `№${number.number}. ${number.title}`
      : `№${number.number} (часть ${number.part})`
  }
  return map
})

const filterKimNumbers = computed(() => {
  if (!filter.exam_year_id) return kimNumbers.value
  return kimNumbers.value.filter((number) => number.exam_year_id === filter.exam_year_id)
})

const formKimNumbers = computed(() =>
  kimNumbers.value.filter((number) => number.exam_year_id === form.exam_year_id),
)

const formTopics = computed(() => {
  const year = examYears.value.find((item) => item.id === form.exam_year_id)
  if (!year) return []
  return topics.value.filter((topic) => topic.exam_id === year.exam_id)
})

function onFilterYearChange() {
  filter.kim_number_id = ''
}

function onFormYearChange() {
  form.kim_number_id = ''
  form.topic_id = ''
}

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

async function loadRefs() {
  try {
    const [years, numbers, topicList, examList] = await Promise.all([
      api('/admin/exam-years?limit=200'),
      api('/admin/kim-numbers?limit=200'),
      api('/admin/topics?limit=200'),
      api('/admin/exams?limit=200')
    ])
    examYears.value = years.items || []
    kimNumbers.value = numbers.items || []
    topics.value = topicList.items || []
    exams.value = examList.items || []
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

onMounted(() => {
  load()
  loadRefs()
})
</script>

<template>
  <h2>Задания</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>

  <form class="admin-form" @submit.prevent="load">
    <label>
      Учебный год
      <select v-model="filter.exam_year_id" @change="onFilterYearChange">
        <option value="">Все годы</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <label>
      Номер КИМ
      <select v-model="filter.kim_number_id">
        <option value="">Все номера</option>
        <option v-for="number in filterKimNumbers" :key="number.id" :value="number.id">{{ kimNumberLabels[number.id] }}</option>
      </select>
    </label>
    <label>
      Статус
      <select v-model="filter.status">
        <option value="">Любой статус</option>
        <option value="draft">Черновик</option>
        <option value="published">Опубликовано</option>
        <option value="archived">Архив</option>
      </select>
    </label>
    <label>
      Поиск
      <input v-model="filter.query" placeholder="Поиск" />
    </label>
    <button type="submit">Фильтр</button>
  </form>

  <h3>{{ editingId ? 'Изменить задание' : 'Новое задание' }}</h3>
  <form class="admin-form" @submit.prevent="save">
    <label>
      Учебный год
      <select v-model="form.exam_year_id" required @change="onFormYearChange">
        <option value="" disabled>Выберите год</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <label>
      Номер КИМ
      <select v-model="form.kim_number_id" required>
        <option value="" disabled>Выберите номер</option>
        <option v-for="number in formKimNumbers" :key="number.id" :value="number.id">{{ kimNumberLabels[number.id] }}</option>
      </select>
    </label>
    <label>
      Тема
      <select v-model="form.topic_id">
        <option value="">Без темы</option>
        <option v-for="topic in formTopics" :key="topic.id" :value="topic.id">{{ topic.code }}. {{ topic.title }}</option>
      </select>
    </label>
    <label>
      Сложность
      <select v-model="form.difficulty">
        <option value="easy">Лёгкое</option>
        <option value="medium">Среднее</option>
        <option value="hard">Сложное</option>
      </select>
    </label>
    <label>
      Источник
      <input v-model="form.source" placeholder="Источник" required />
    </label>
    <label>
      Ссылка на источник
      <input v-model="form.source_ref" placeholder="Ссылка на источник" />
    </label>
    <label>
      Отображаемый ответ
      <input v-model="form.answer_display" placeholder="Отображаемый ответ" />
    </label>
    <label>
      answer_spec JSON
      <textarea v-model="form.answer_spec" placeholder="answer_spec JSON"></textarea>
    </label>
    <button type="submit">Сохранить</button>
    <button v-if="editingId" type="button" class="secondary" @click="resetForm">Отмена</button>
  </form>
  <div class="admin-field">
    <span>Условие</span>
    <Editor v-model="form.statement_md" />
  </div>

  <h3>Импорт</h3>
  <div class="admin-field">
    <span>JSON для импорта</span>
    <textarea v-model="importText" placeholder='[{"exam_year_id":"...","kim_number_id":"...","difficulty":"easy","statement_md":"...","answer_spec":{"kind":"number","accepted":["1"]},"source":"seed"}]'></textarea>
  </div>
  <button @click="importTasks">Импортировать</button>

  <h3>Изображение задания</h3>
  <form class="admin-form" @submit.prevent="upload">
    <label>
      ID задания
      <input v-model="uploadTaskId" placeholder="ID задания" required />
    </label>
    <label>
      Файл изображения
      <input type="file" accept="image/png,image/jpeg,image/gif,image/webp" @change="uploadFile = $event.target.files[0]" />
    </label>
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
