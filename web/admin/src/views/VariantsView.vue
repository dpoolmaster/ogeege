<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const examYears = ref([])
const exams = ref([])
const error = ref('')
const manualTasks = ref('[]')
const rules = ref('[]')
const form = reactive({ exam_year_id: '', title: 'Вариант', kind: 'mock', duration_minutes: 235 })
const details = ref(null)
const detailTasks = ref([])
const detailLoading = ref(false)

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

async function loadRefs() {
  try {
    const [years, examList] = await Promise.all([
      api('/admin/exam-years?limit=200'),
      api('/admin/exams?limit=200')
    ])
    examYears.value = years.items || []
    exams.value = examList.items || []
    if (!form.exam_year_id && examYears.value.length > 0) form.exam_year_id = examYears.value[0].id
  } catch (err) {
    error.value = err.message
  }
}

async function load() {
  try {
    const data = await api('/admin/variants?limit=100')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/variants', {
      method: 'POST',
      body: JSON.stringify({ ...form, tasks: JSON.parse(manualTasks.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function generate() {
  error.value = ''
  try {
    await api('/admin/variants/generate', {
      method: 'POST',
      body: JSON.stringify({ ...form, rules: JSON.parse(rules.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/variants/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function toggleDetails(item) {
  if (details.value?.id === item.id) {
    details.value = null
    detailTasks.value = []
    return
  }
  error.value = ''
  detailLoading.value = true
  try {
    const variant = await api(`/admin/variants/${item.id}`)
    const tasks = await Promise.all((variant.tasks || []).map(async (position) => {
      try {
        const task = await api(`/admin/tasks/${position.task_id}`)
        return { ...position, ...task }
      } catch (err) {
        return { ...position, statement_md: 'Не удалось загрузить задание' }
      }
    }))
    details.value = variant
    detailTasks.value = tasks
  } catch (err) {
    error.value = err.message
  } finally {
    detailLoading.value = false
  }
}

onMounted(() => {
  load()
  loadRefs()
})
</script>

<template>
  <h2>Варианты</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <label>
      Учебный год
      <select v-model="form.exam_year_id" required>
        <option value="" disabled>Выберите год</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <label>
      Название
      <input v-model="form.title" placeholder="Например, Пробник №1" required />
    </label>
    <label>
      Вид варианта
      <select v-model="form.kind">
        <option value="diagnostic">Диагностика</option>
        <option value="practice">Практика</option>
        <option value="mock">Пробник</option>
      </select>
    </label>
    <label>
      Длительность, минуты
      <input v-model.number="form.duration_minutes" type="number" min="1" required />
    </label>
    <label>
      Задания вручную (JSON)
      <textarea v-model="manualTasks" placeholder='[{"task_id":"...","points":1}]'></textarea>
    </label>
    <button type="submit">Создать вручную</button>
  </form>
  <form class="admin-form" @submit.prevent="generate">
    <label>
      Правила генерации (JSON)
      <textarea v-model="rules" placeholder='[{"kim_number_id":"...","count":1,"difficulty":"easy","topic_id":null}]'></textarea>
    </label>
    <button type="submit">Сгенерировать</button>
  </form>
  <table>
    <thead><tr><th>Название</th><th>Вид</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.title }}</td>
        <td>{{ item.kind }}</td>
        <td>{{ item.status }}</td>
        <td class="admin-actions">
          <button type="button" class="secondary" @click="toggleDetails(item)">
            {{ details?.id === item.id ? 'Скрыть состав' : 'Состав' }}
          </button>
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
        </td>
      </tr>
    </tbody>
  </table>
  <p v-if="detailLoading">Загрузка состава…</p>
  <div v-else-if="details">
    <h3>Состав варианта: {{ details.title }}</h3>
    <p>
      Вид: {{ details.kind }}, длительность: {{ details.duration_minutes }} мин,
      статус: {{ details.status }}, заданий: {{ detailTasks.length }}
    </p>
    <table>
      <thead><tr><th>№</th><th>Задание</th><th>Баллы</th><th>Статус задания</th></tr></thead>
      <tbody>
        <tr v-for="task in detailTasks" :key="task.task_id">
          <td>{{ task.position }}</td>
          <td class="variant-task-statement">{{ task.statement_md }}</td>
          <td>{{ task.points }}</td>
          <td>{{ task.status || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
