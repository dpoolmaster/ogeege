<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const examYears = ref([])
const kimNumbers = ref([])
const exams = ref([])
const error = ref('')
const editingId = ref('')
const criteria = ref('[]')
const form = reactive({ exam_year_id: '', kim_number_id: '' })

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

const formKimNumbers = computed(() =>
  kimNumbers.value.filter((number) => number.exam_year_id === form.exam_year_id),
)

function onFormYearChange() {
  form.kim_number_id = ''
}

async function load() {
  try {
    const data = await api('/admin/rubrics?limit=200')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function loadRefs() {
  try {
    const [years, numbers, examList] = await Promise.all([
      api('/admin/exam-years?limit=200'),
      api('/admin/kim-numbers?limit=200'),
      api('/admin/exams?limit=200')
    ])
    examYears.value = years.items || []
    kimNumbers.value = numbers.items || []
    exams.value = examList.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/rubrics', {
      method: 'POST',
      body: JSON.stringify({ ...form, criteria: JSON.parse(criteria.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

function edit(item) {
  editingId.value = item.id
  criteria.value = JSON.stringify(item.criteria)
}

async function update() {
  error.value = ''
  try {
    await api(`/admin/rubrics/${editingId.value}`, {
      method: 'PUT',
      body: JSON.stringify({ criteria: JSON.parse(criteria.value) })
    })
    editingId.value = ''
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/rubrics/${item.id}/publish`, { method: 'POST' })
    await load()
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
  <h2>Рубрики</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
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
      Критерии (JSON)
      <textarea v-model="criteria" placeholder='[{"code":"C1","description":"...","max_points":2}]'></textarea>
    </label>
    <button type="submit">Создать</button>
  </form>
  <button v-if="editingId" @click="update">Сохранить новую версию</button>
  <table>
    <thead><tr><th>Версия</th><th>Учебный год</th><th>Номер КИМ</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.version }}</td>
        <td>{{ examYearLabels[item.exam_year_id] || item.exam_year_id }}</td>
        <td>{{ kimNumberLabels[item.kim_number_id] || item.kim_number_id }}</td>
        <td>{{ item.status }}</td>
        <td class="admin-actions">
          <button @click="edit(item)">Изменить</button>
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
